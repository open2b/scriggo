//
// Copyright (c) 2016-2017 Open2b Software Snc. All Rights Reserved.
//

package template

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"open2b/template/ast"
)

// Parse esegue il parsing di src e ne restituisce l'albero non espanso.
func Parse(src []byte) (*ast.Tree, error) {

	// crea il lexer
	var lex = newLexer(src)

	// albero risultato del parsing
	var tree = ast.NewTree("", nil)

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

			// var or identifier
			case tokenVar, tokenIdentifier:
				var isVar = tok.typ == tokenVar
				if isVar {
					// identifier
					tok, ok = <-lex.tokens
					if !ok {
						return nil, lex.err
					}
					if tok.typ != tokenIdentifier {
						return nil, fmt.Errorf("unexpected %s, expecting identifier at %d", tok, tok.pos)
					}
				}
				var ident = ast.NewIdentifier(tok.pos, string(tok.txt))
				// assignment
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenAssignment {
					return nil, fmt.Errorf("unexpected %s, expecting assignment at %d", tok, tok.pos)
				}
				// expression
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
				pos.End = tok.pos.End
				if isVar {
					node = ast.NewVar(pos, ident, expr)
				} else {
					node = ast.NewAssignment(pos, ident, expr)
				}
				addChild(parent, node)

			// for
			case tokenFor:
				var index, ident *ast.Identifier
				// index
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenIdentifier {
					return nil, fmt.Errorf("unexpected %s, expecting identifier at %d", tok, tok.pos)
				}
				index = ast.NewIdentifier(tok.pos, string(tok.txt))
				// "," or "in"
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				switch tok.typ {
				case tokenComma:
					// syntax: for index, ident in expr
					tok, ok = <-lex.tokens
					if !ok {
						return nil, lex.err
					}
					if tok.typ != tokenIdentifier {
						return nil, fmt.Errorf("unexpected %s, expecting identifier at %d", tok, tok.pos)
					}
					ident = ast.NewIdentifier(tok.pos, string(tok.txt))
					// "in"
					tok, ok = <-lex.tokens
					if !ok {
						return nil, lex.err
					}
					if tok.typ != tokenIn {
						return nil, fmt.Errorf("unexpected %s, expecting \"in\" at %d", tok, tok.pos)
					}
				case tokenIn:
					// syntax: for ident in expr
					ident = index
					index = nil
				default:
					return nil, fmt.Errorf("unexpected %s, expecting comma or \"in\" at %d", tok, tok.pos)
				}
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
				pos.End = tok.pos.End
				node = ast.NewFor(pos, index, ident, expr, nil)
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
				pos.End = tok.pos.End
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
				pos.End = tok.pos.End
				node = ast.NewShow(pos, expr, parserContext(ctx))
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
				pos.End = tok.pos.End
				node = ast.NewExtend(pos, path, nil)
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
				pos.End = tok.pos.End
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
				pos.End = tok.pos.End
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
			tok.pos.End = tok2.pos.End
			var node = ast.NewShow(tok.pos, expr, parserContext(tok.ctx))
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

// ReadFunc ritorna un albero dato il path.
type ReadFunc func(path string) (*ast.Tree, error)

// FileReader fornisce una ReadFunc che legge i sorgenti dai
// file nella directory dir.
func FileReader(dir string) ReadFunc {
	return func(path string) (*ast.Tree, error) {
		src, err := ioutil.ReadFile(filepath.Join(dir, path))
		if err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, err
		}
		tree, err := Parse(src)
		if err != nil {
			return nil, err
		}
		tree.Path = path
		return tree, nil
	}
}

// CacheReader restituisce una ReadFunc che mantiene in cache
// gli alberi letti con read.
func CacheReader(read ReadFunc) ReadFunc {
	var trees = map[string]*ast.Tree{}
	return func(path string) (*ast.Tree, error) {
		var err error
		tree, ok := trees[path]
		if !ok {
			tree, err = read(path)
			if err != nil && tree != nil {
				trees[path] = tree
			}
		}
		return tree, err
	}
}

type Parser struct {
	read ReadFunc
}

func NewParser(read ReadFunc) *Parser {
	return &Parser{read}
}

// Parse ritorna l'albero espanso di path il quale deve essere assoluto.
// Legge l'albero tramite la funzione read ed espande i nodi Extend e
// Include se presenti.
func (p *Parser) Parse(path string) (*ast.Tree, error) {

	var err error

	// path deve essere valido e assoluto
	if !isValidFilePath(path) {
		return nil, fmt.Errorf("template: invalid path %q", path)
	}
	if path[0] != '/' {
		return nil, fmt.Errorf("template: path %q is not absolute", path)
	}

	// pulisce path rimuovendo ".."
	path, err = toAbsolutePath(path[:1], path[1:])
	if err != nil {
		return nil, err
	}

	tree, err := p.read(path)
	if err != nil {
		return nil, err
	}
	if tree == nil {
		return nil, fmt.Errorf("template: path %q does not exist", path)
	}

	var dir = path[:strings.LastIndexByte(path, '/')+1]

	// determina il nodo extend se presente
	var extend = getExtendNode(tree)

	if extend != nil && extend.Tree == nil {
		path, err := toAbsolutePath(dir, extend.Path)
		if err != nil {
			return nil, err
		}
		tr, err := p.read(path)
		if err != nil {
			return nil, err
		}
		// verifica che l'albero di extend non contenga a sua volta extend
		if ex := getExtendNode(tr); ex != nil {
			return nil, fmt.Errorf("template: extended document contains extend at %q", ex.Pos())
		}
		// espande gli includes
		var d = path[:strings.LastIndexByte(path, '/')+1]
		err = p.expandIncludes(tr.Nodes, d)
		if err != nil {
			return nil, err
		}
		extend.Tree = tr
	}

	// espande gli includes
	err = p.expandIncludes(tree.Nodes, dir)
	if err != nil {
		return nil, err
	}

	return tree, nil
}

// expandIncludes espande gli include dell'albero tree chiamando read
// e anteponendo dir al path passato come argomento.
func (p *Parser) expandIncludes(nodes []ast.Node, dir string) error {
	var err error
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.For:
			p.expandIncludes(n.Nodes, dir)
		case *ast.If:
			p.expandIncludes(n.Nodes, dir)
		case *ast.Region:
			p.expandIncludes(n.Nodes, dir)
		case *ast.Include:
			if n.Tree == nil {
				var path = n.Path
				if path[0] != '/' {
					path, err = toAbsolutePath(dir, path)
					if err != nil {
						return err
					}
				}
				include, err := p.read(path)
				if err != nil {
					return err
				}
				var d = path[:strings.LastIndexByte(path, '/')+1]
				err = p.expandIncludes(include.Nodes, d)
				if err != nil {
					return err
				}
				n.Tree = include
			}
		}
	}
	return nil
}

func addChild(parent ast.Node, node ast.Node) {
	switch n := parent.(type) {
	case *ast.Tree:
		n.Nodes = append(n.Nodes, node)
	case *ast.Region:
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

// getExtendNode ritorna il nodo Extend di un albero.
// Se il nodo non è presente ritorna nil.
func getExtendNode(tree *ast.Tree) *ast.Extend {
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
