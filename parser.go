//
// Copyright (c) 2016 Open2b Software Snc. All Rights Reserved.
//

package template

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"open2b/template/ast"
)

// ReadFunc è chiamata dal metodo Parse per leggere il testo di un path.
// Non viene mai chiamata per più di una volta per uno stesso path.
type ReadFunc func(path string) (io.Reader, error)

// TransformFunc applica delle trasformazioni ad un albero non espanso.
type TransformFunc func(tree *ast.Tree, path string) error

// Parse esegue il parsing del testo letto da text e ne restituisce l'albero
// non espanso. Per avere invece l'albero espanso utilizzare il metodo Parse.
func Parse(text io.Reader) (*ast.Tree, error) {
	var buf bytes.Buffer
	var err = readBufferAll(text, &buf)
	if err != nil {
		return nil, err
	}
	return parse(buf.Bytes())
}

// Parser è un parser.
type Parser struct {
	read      ReadFunc
	transform TransformFunc
	trees     map[string]*ast.Tree
	buf       *bytes.Buffer
}

// NewParser crea una nuovo parser. I sorgenti dei template sono letti
// tramite la funzione read.
func NewParser(read ReadFunc) *Parser {
	return &Parser{
		read:  read,
		trees: map[string]*ast.Tree{},
		buf:   &bytes.Buffer{},
	}
}

// Parse ritorna l'albero espanso di path il quale deve essere assoluto. Legge
// il sorgente tramite la funzione read ed espande i nodi Extend, Region e
// Include se presenti.
func (p *Parser) Parse(path string) (*ast.Tree, error) {

	// path deve essere valido e assoluto
	if !isValidFilePath(path) {
		return nil, fmt.Errorf("template: invalid path %q", path)
	}
	if path[0] != '/' {
		return nil, fmt.Errorf("template: path %q is not absolute", path)
	}

	// base per i path relativi
	var base = path[:strings.LastIndexByte(path, '/')+1]

	tree, err := p.tree(path)
	if err != nil {
		return nil, err
	}

	// determina il path di extend se presente
	var extend = getExtend(tree)

	// determina l'albero da visitare e le eventuali regions
	var root = tree
	var regions map[string]*ast.Region

	if extend != nil {
		regions = map[string]*ast.Region{}
		for _, node := range tree.Nodes {
			if n, ok := node.(*ast.Region); ok {
				regions[n.Name] = n
			}
		}
		root, err = p.tree(extend.Path)
		if err != nil {
			return nil, err
		}
		// root non deve contenere extend
		if ex := getExtend(root); ex != nil {
			return nil, fmt.Errorf("template: extended document can not contains extend at %q", path, ex.Pos())
		}
		// espande le region
		for _, node := range root.Nodes {
			if n, ok := node.(*ast.Region); ok {
				if regions == nil {
					n.Nodes = []ast.Node{}
				} else if region, ok := regions[n.Name]; ok {
					n.Nodes = region.Nodes
				} else {
					n.Nodes = []ast.Node{}
				}
			}
		}
	}

	// espande gli includes
	err = p.expandIncludes(root, base)
	if err != nil {
		return nil, err
	}

	return root, nil
}

// tree restituisce l'albero non espanso relativo a path il quale deve essere
// assoluto.
func (p *Parser) tree(path string) (*ast.Tree, error) {
	tree, ok := p.trees[path]
	if !ok {
		text, err := p.read(path)
		if err != nil {
			return nil, err
		}
		defer p.buf.Reset()
		err = readBufferAll(text, p.buf)
		if err != nil {
			return nil, err
		}
		tree, err = parse(p.buf.Bytes())
		if err != nil {
			if e, ok := err.(*SyntaxError); ok {
				e.path = path
			}
			return nil, err
		}
		p.trees[path] = tree
	}
	return ast.CloneTree(tree), nil
}

// expandIncludes espande gli include nel nodo node utilizzando base come
// base per i path relativi.
func (p *Parser) expandIncludes(node ast.Node, base string) error {
	var nodes = []ast.Node{node}
	for len(nodes) > 0 {
		var node = nodes[len(nodes)-1]
		nodes = nodes[:len(nodes)-1]
		switch n := node.(type) {
		case *ast.For:
			nodes = append(nodes, n.Nodes...)
		case *ast.If:
			nodes = append(nodes, n.Nodes...)
		case *ast.Include:
			var path = n.Path
			if path[0] != '/' {
				path = base + path
			}
			include, err := p.tree(path)
			if err != nil {
				return err
			}
			// base per i path relativi
			var ba = path[:strings.LastIndexByte(path, '/')+1]
			for _, in := range include.Nodes {
				err = p.expandIncludes(in, ba)
				if err != nil {
					return err
				}
			}
			n.Nodes = include.Nodes
		}
	}
	return nil
}

// parse esegue il parsing del testo text e ne restituisce l'albero non espanso.
func parse(text []byte) (*ast.Tree, error) {

	// crea il lexer
	var lex = newLexer(text)

	// albero risultato del parsing
	var tree = ast.NewTree(nil)

	// antenati dalla radice fino al genitore
	var ancestors = []ast.Node{tree}

	// legge tutti i token
	for tok := range lex.tokens {

		// il genitore è sempre l'ultimo antenato
		parent := ancestors[len(ancestors)-1]

		switch tok.typ {

		// EOF
		case tokenEOF:
			break

		// Text
		case tokenText:
			addChild(parent, ast.NewText(tok.pos, string(tok.txt)))

		// {%
		case tokenStartStatement:

			var node ast.Node

			var pos = tok.pos
			var ctx = tok.ctx

			var ok bool
			tok, ok = <-lex.tokens
			if !ok {
				return nil, lex.err
			}

			switch tok.typ {

			// for
			case tokenFor:
				expr, tok, err := parseExpr(lex)
				if err != nil {
					return nil, err
				}
				if expr == nil {
					return nil, fmt.Errorf("expecting expression at %d", tok.pos)
				}
				if tok.typ != tokenEndStatement {
					return nil, fmt.Errorf("unexpected %s, expecting %%} at %d", tok, tok.pos)
				}
				node = ast.NewFor(pos, expr, nil)
				addChild(parent, node)
				ancestors = append(ancestors, node)

			// if
			case tokenIf:
				expr, tok, err := parseExpr(lex)
				if err != nil {
					return nil, err
				}
				if expr == nil {
					return nil, fmt.Errorf("expecting expression at %d", tok.pos)
				}
				if tok.typ != tokenEndStatement {
					return nil, fmt.Errorf("unexpected %s, expecting %%} at %d", tok, tok.pos)
				}
				node = ast.NewIf(pos, expr, nil)
				addChild(parent, node)
				ancestors = append(ancestors, node)

			// show
			case tokenShow:
				expr, tok, err := parseExpr(lex)
				if err != nil {
					return nil, err
				}
				if expr == nil {
					return nil, fmt.Errorf("expecting expression at %d", tok.pos)
				}
				if tok.typ != tokenEndStatement {
					return nil, fmt.Errorf("unexpected %s, expecting %%} at %d", tok, tok.pos)
				}
				node = ast.NewShow(pos, expr, nil, parserContext(ctx))
				addChild(parent, node)
				ancestors = append(ancestors, node)

			// extend
			case tokenExtend:
				path, err := parsePath(lex)
				if err != nil {
					return nil, err
				}
				tok, ok := <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenEndStatement {
					return nil, fmt.Errorf("unexpected %s, expecting %%} at %d", tok, tok.pos)
				}
				node = ast.NewExtend(pos, path)
				addChild(parent, node)

			// region
			case tokenRegion:
				name, err := parseString(lex)
				if err != nil {
					return nil, err
				}
				if name == "" {
					return nil, fmt.Errorf("region name can not be empty at %d", pos)
				}
				tok, ok := <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenEndStatement {
					return nil, fmt.Errorf("unexpected %s, expecting %%} at %d", tok, tok.pos)
				}
				node = ast.NewRegion(pos, name, nil)
				addChild(parent, node)
				ancestors = append(ancestors, node)

			// include
			case tokenInclude:
				path, err := parsePath(lex)
				if err != nil {
					return nil, err
				}
				tok, ok := <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenEndStatement {
					return nil, fmt.Errorf("unexpected %s, expecting %%} at %d", tok, tok.pos)
				}
				node = ast.NewInclude(pos, path, nil)
				addChild(parent, node)

			// // snippet
			// case tokenSnippet:
			// 	expr, tok, err := parseSnippetName(lex)
			// 	if err != nil {
			// 		return nil, err
			// 	}
			// 	node = ast.NewSnippet(pos, prefix, name)

			// end
			case tokenEnd:
				tok, ok := <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenEndStatement {
					return nil, fmt.Errorf("unexpected %s, expecting %%} at %d", tok, tok.pos)
				}
				if len(ancestors) == 1 {
					return nil, fmt.Errorf("unexpected end statement at %d", pos)
				}
				ancestors = ancestors[:len(ancestors)-1]

			default:
				return nil, fmt.Errorf("unexpected %s, expecting for, if, show, extend, region or end at %d", tok, tok.pos)

			}

		// {{ }}
		case tokenStartShow:
			expr, tok2, err := parseExpr(lex)
			if err != nil {
				return nil, err
			}
			if expr == nil {
				return nil, fmt.Errorf("expecting expression at %d", tok2.pos)
			}
			if tok2.typ != tokenEndShow {
				return nil, fmt.Errorf("unexpected %s, expecting }} at %d", tok2, tok2.pos)
			}
			var node = ast.NewShow(tok.pos, expr, nil, parserContext(tok.ctx))
			addChild(parent, node)

		default:
			return nil, fmt.Errorf("unexpected %s at %d", tok, tok.pos)

		}

	}

	if lex.err != nil {
		return nil, lex.err
	}

	return tree, nil
}

// readBufferAll copia nel buffer buf tutti i bytes letti da r.
func readBufferAll(r io.Reader, buf *bytes.Buffer) (err error) {
	defer func() {
		// converte il panic ErrTooLarge in un errore da ritornare
		if e := recover(); e != nil {
			if e2, ok := e.(error); ok && e2 == bytes.ErrTooLarge {
				err = e2
			} else {
				panic(e)
			}
		}
	}()
	_, err = buf.ReadFrom(r)
	return err
}

func addChild(parent ast.Node, node ast.Node) {
	switch n := parent.(type) {
	case *ast.Tree:
		n.Nodes = append(n.Nodes, node)
	case *ast.Show:
		n.Text = node.(*ast.Text)
	case *ast.Region:
		n.Nodes = append(n.Nodes, node)
	case *ast.Include:
		n.Nodes = append(n.Nodes, node)
	case *ast.For:
		n.Nodes = append(n.Nodes, node)
	case *ast.If:
		n.Nodes = append(n.Nodes, node)
	}
}

func parserContext(ctx context) ast.Context {
	switch ctx {
	case contextHTML:
		return ast.ContextHTML
	case contextAttribute:
		return ast.ContextAttribute
	case contextScript:
		return ast.ContextScript
	case contextStyle:
		return ast.ContextScript
	default:
		panic("invalid context type")
	}
}

// getExtend ritorna il nodo Extend di un albero.
// Se il nodo non è presente ritorna nil.
func getExtend(tree *ast.Tree) *ast.Extend {
	if len(tree.Nodes) == 0 {
		return nil
	}
	if node, ok := tree.Nodes[0].(*ast.Extend); ok {
		return node
	}
	if len(tree.Nodes) > 1 {
		if node, ok := tree.Nodes[1].(*ast.Extend); ok {
			return node
		}
	}
	return nil
}
