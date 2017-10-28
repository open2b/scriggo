//
// Copyright (c) 2016-2017 Open2b Software Snc. All Rights Reserved.
//

// Package parser fornisce i metodi per parsare i sorgenti dei
// templates e ritornarne gli alberi.
package parser

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"open2b/template/ast"
)

var (
	// ErrInvalid è ritornato dal metodo Parse quando il parametro path non è valido.
	ErrInvalid = errors.New("template/parser: invalid argument")

	// ErrNotExist è ritornato dal metodo Parse e da una ReadFunc quando il path non esiste.
	ErrNotExist = errors.New("template/parser: path does not exist")
)

type Error struct {
	Path string
	Pos  ast.Position
	Err  error
}

func (e *Error) Error() string {
	return fmt.Sprintf("template: %s at %q %s", e.Err, e.Path, e.Pos)
}

// Parse esegue il parsing di src e ne restituisce l'albero non espanso.
func Parse(src []byte) (*ast.Tree, error) {

	// crea il lexer
	var lex = newLexer(src)

	// albero risultato del parsing
	var tree = ast.NewTree("", nil)

	// antenati dalla radice fino al genitore
	var ancestors = []ast.Node{tree}

	// linea corrente
	var line = 0

	// primo nodo Text della linea corrente
	var firstText *ast.Text

	// indica se è presente un token nella linea corrente per cui sarebbe
	// possibile tagliare gli spazi iniziali e finali
	var cutSpacesToken bool

	// numero di token, non Text, nella linea corrente
	var tokensInLine = 0

	// last byte index
	var end = len(src) - 1

	// legge tutti i token
	for tok := range lex.tokens {

		var text *ast.Text
		if tok.typ == tokenText {
			text = ast.NewText(tok.pos, string(tok.txt))
		}

		if line < tok.lin || tok.pos.End == end {
			if cutSpacesToken && tokensInLine == 1 {
				cutSpaces(firstText, text)
			}
			line = tok.lin
			firstText = text
			cutSpacesToken = false
			tokensInLine = 0
		}

		// il genitore è sempre l'ultimo antenato
		parent := ancestors[len(ancestors)-1]

		switch tok.typ {

		// EOF
		case tokenEOF:

		// Text
		case tokenText:
			addChild(parent, text)

		// {%
		case tokenStartStatement:

			tokensInLine++

			var node ast.Node

			var pos = tok.pos
			var ctx = tok.ctx

			var err error
			var expr ast.Expression

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
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting identifier", tok)}
					}
				}
				var ident = ast.NewIdentifier(tok.pos, string(tok.txt))
				// assignment
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenAssignment {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting assignment", tok)}
				}
				// expression
				expr, tok, err = parseExpr(lex)
				if err != nil {
					return nil, err
				}
				if expr == nil {
					return nil, &Error{"", *tok.pos, fmt.Errorf("expecting expression")}
				}
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				pos.End = tok.pos.End
				if isVar {
					node = ast.NewVar(pos, ident, expr)
				} else {
					node = ast.NewAssignment(pos, ident, expr)
				}
				addChild(parent, node)
				cutSpacesToken = true

			// for
			case tokenFor:
				var index, ident *ast.Identifier
				// index
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenIdentifier {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting identifier", tok)}
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
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting identifier", tok)}
					}
					ident = ast.NewIdentifier(tok.pos, string(tok.txt))
					// "in"
					tok, ok = <-lex.tokens
					if !ok {
						return nil, lex.err
					}
					if tok.typ != tokenIn {
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting \"in\"", tok)}
					}
				case tokenIn:
					// syntax: for ident in expr
					ident = index
					index = nil
				default:
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting comma or \"in\"", tok)}
				}
				expr, tok, err = parseExpr(lex)
				if err != nil {
					return nil, err
				}
				if expr == nil {
					return nil, &Error{"", *tok.pos, fmt.Errorf("expecting expression")}
				}
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				pos.End = tok.pos.End
				node = ast.NewFor(pos, index, ident, expr, nil)
				addChild(parent, node)
				ancestors = append(ancestors, node)
				cutSpacesToken = true

			// if
			case tokenIf:
				expr, tok, err = parseExpr(lex)
				if err != nil {
					return nil, err
				}
				if expr == nil {
					return nil, &Error{"", *tok.pos, fmt.Errorf("expecting expression")}
				}
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				pos.End = tok.pos.End
				node = ast.NewIf(pos, expr, nil, nil)
				addChild(parent, node)
				ancestors = append(ancestors, node)
				cutSpacesToken = true

			// else
			case tokenElse:
				parent := ancestors[len(ancestors)-1]
				if p, ok := parent.(*ast.If); ok && p.Else == nil {
					p.Else = []ast.Node{}
				} else {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end, for, if, show or include", tok)}
				}
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				cutSpacesToken = true

			// show
			case tokenShow:
				expr, tok, err = parseExpr(lex)
				if err != nil {
					return nil, err
				}
				if expr == nil {
					return nil, &Error{"", *tok.pos, fmt.Errorf("expecting expression")}
				}
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				pos.End = tok.pos.End
				node = ast.NewShow(pos, expr, parserContext(ctx))
				addChild(parent, node)
				ancestors = append(ancestors, node)
				cutSpacesToken = true

			// extend
			case tokenExtend:
				path, err := parsePath(lex)
				if err != nil {
					return nil, err
				}
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
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
					return nil, &Error{"", *pos, fmt.Errorf("region name can not be empty")}
				}
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				pos.End = tok.pos.End
				node = ast.NewRegion(pos, name, nil)
				addChild(parent, node)
				ancestors = append(ancestors, node)
				cutSpacesToken = true

			// include
			case tokenInclude:
				path, err := parsePath(lex)
				if err != nil {
					return nil, err
				}
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				pos.End = tok.pos.End
				node = ast.NewInclude(pos, path, nil)
				addChild(parent, node)
				cutSpacesToken = true

			// end
			case tokenEnd:
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenEndStatement {
					parent := ancestors[len(ancestors)-1]
					switch parent.(type) {
					case *ast.For:
						if tok.typ != tokenFor {
							return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting for or %%}", tok)}
						}
					case *ast.If:
						if tok.typ != tokenIf {
							return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting if or %%}", tok)}
						}
					case *ast.Show:
						if tok.typ != tokenShow {
							return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting show or %%}", tok)}
						}
					case *ast.Region:
						if tok.typ != tokenRegion {
							return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting region or %%}", tok)}
						}
					default:
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting for, if, show, extend, region or %%}", tok)}
					}
					tok, ok = <-lex.tokens
					if !ok {
						return nil, lex.err
					}
					if tok.typ != tokenEndStatement {
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
					}
				}
				ancestors = ancestors[:len(ancestors)-1]
				cutSpacesToken = true

			default:
				return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting for, if, show, extend, region or end", tok)}

			}

		// {{ }}
		case tokenStartShow:
			tokensInLine++
			expr, tok2, err := parseExpr(lex)
			if err != nil {
				return nil, err
			}
			if expr == nil {
				return nil, &Error{"", *tok2.pos, fmt.Errorf("expecting expression")}
			}
			if tok2.typ != tokenEndShow {
				return nil, &Error{"", *tok2.pos, fmt.Errorf("unexpected %s, expecting }}", tok2)}
			}
			tok.pos.End = tok2.pos.End
			var node = ast.NewShow(tok.pos, expr, parserContext(tok.ctx))
			addChild(parent, node)

		default:
			return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s", tok)}

		}

	}

	if lex.err != nil {
		return nil, lex.err
	}

	return tree, nil
}

// ReadFunc ritorna un albero dato il path.
// Se path non esiste ritorna l'errore ErrNotExist.
type ReadFunc func(path string) (*ast.Tree, error)

// FileReader fornisce una ReadFunc che legge i sorgenti dai
// file nella directory dir.
func FileReader(dir string) ReadFunc {
	return func(path string) (*ast.Tree, error) {
		src, err := ioutil.ReadFile(filepath.Join(dir, path))
		if err != nil {
			if os.IsNotExist(err) {
				return nil, ErrNotExist
			}
			return nil, err
		}
		tree, err := Parse(src)
		if err != nil {
			if err2, ok := err.(*Error); ok {
				err2.Path = path
			}
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
	var mux sync.Mutex
	return func(path string) (*ast.Tree, error) {
		var err error
		mux.Lock()
		tree, ok := trees[path]
		mux.Unlock()
		if !ok {
			tree, err = read(path)
			if err != nil {
				mux.Lock()
				trees[path] = tree
				mux.Unlock()
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
	if !isValidFilePath(path) || path[0] != '/' {
		return nil, ErrInvalid
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

	tree.Lock()
	defer tree.Unlock()

	// verifica se è già stato espanso
	if tree.IsExpanded {
		return tree, nil
	}

	var dir = path[:strings.LastIndexByte(path, '/')+1]

	// determina il nodo extend se presente
	var extend = getExtendNode(tree)

	if extend != nil && extend.Tree == nil {
		var exPath string
		if extend.Path[0] == '/' {
			// pulisce il path rimuovendo ".."
			exPath, err = toAbsolutePath("/", extend.Path[1:])
		} else {
			// determina il path assoluto
			exPath, err = toAbsolutePath(dir, extend.Path)
		}
		if err != nil {
			return nil, err
		}
		var tr *ast.Tree
		tr, err = p.read(exPath)
		if err != nil {
			if err == ErrNotExist {
				err = &Error{tree.Path, *(extend.Pos()), fmt.Errorf("extend path %q does not exist", exPath)}
			}
			return nil, err
		}
		// verifica che l'albero di extend non contenga a sua volta extend
		if ex := getExtendNode(tr); ex != nil {
			return nil, &Error{tree.Path, *(ex.Pos()), fmt.Errorf("extend document contains extend")}
		}
		// verifica se è stato espanso
		tr.Lock()
		if !tr.IsExpanded {
			// espande gli includes
			var d = exPath[:strings.LastIndexByte(exPath, '/')+1]
			err = p.expandIncludes(tr.Nodes, d)
			if err == nil {
				tr.IsExpanded = true
			}
		}
		tr.Unlock()
		if err != nil {
			if err2, ok := err.(*Error); ok && err2.Path == "" {
				err2.Path = exPath
			}
			return nil, err
		}
		extend.Tree = tr
	}

	// espande gli includes
	err = p.expandIncludes(tree.Nodes, dir)
	if err != nil {
		if err2, ok := err.(*Error); ok && err2.Path == "" {
			err2.Path = path
		}
		return nil, err
	}

	tree.IsExpanded = true

	return tree, nil
}

// expandIncludes espande gli include dell'albero tree chiamando read
// e anteponendo dir al path passato come argomento.
func (p *Parser) expandIncludes(nodes []ast.Node, dir string) error {
	var err error
	for _, node := range nodes {
		switch n := node.(type) {
		case *ast.For:
			err = p.expandIncludes(n.Nodes, dir)
		case *ast.If:
			err = p.expandIncludes(n.Then, dir)
			if err == nil {
				err = p.expandIncludes(n.Else, dir)
			}
		case *ast.Region:
			err = p.expandIncludes(n.Nodes, dir)
		case *ast.Include:
			if n.Tree == nil {
				var path = n.Path
				if path[0] != '/' {
					path, err = toAbsolutePath(dir, path)
					if err != nil {
						return err
					}
				}
				var include *ast.Tree
				include, err = p.read(path)
				if err != nil {
					if err == ErrNotExist {
						err = &Error{"", *(n.Pos()), fmt.Errorf("include path %q does not exist", path)}
					}
					return err
				}
				// se non è espanso lo espande
				include.Lock()
				if !include.IsExpanded {
					var d = path[:strings.LastIndexByte(path, '/')+1]
					err = p.expandIncludes(include.Nodes, d)
				}
				include.Unlock()
				if err != nil {
					if err2, ok := err.(*Error); ok {
						if strings.HasSuffix(err2.Error(), "does not exist") {
							err2.Path = path
						}
					}
					return err
				}
				n.Tree = include
			}
		}
		if err != nil {
			return err
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
		if n.Else == nil {
			n.Then = append(n.Then, node)
		} else {
			n.Else = append(n.Else, node)
		}
	}
}

func parserContext(ctx context) ast.Context {
	switch ctx {
	case contextHTML:
		return ast.ContextHTML
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

// cutSpaces taglia gli spazi iniziali e finali da una linea dove first e last
// sono rispettivamente il nodo Text iniziale e finale.
func cutSpaces(first, last *ast.Text) {
	var firstCut int
	if first != nil {
		// perché gli spazi possano essere tagliati, first.Text deve contenere
		// solo ' ' e '\t', oppure dopo l'ultimo '\n' deve contenere solo ' ' e '\t'.
		txt := first.Text
		for i := len(txt) - 1; i >= 0; i-- {
			c := txt[i]
			if c == '\n' {
				firstCut = i + 1
				break
			}
			if c == '\r' {
				// '\n' può essere seguito da '\r'
				if i > 0 && txt[i-1] == '\n' {
					firstCut = i + 1
					break
				}
				return
			}
			if c != ' ' && c != '\t' {
				return
			}
		}
	}
	if last != nil {
		// perché gli spazi possano essere tagliati, last.Text deve contenere
		// solo ' ' e '\t', oppure prima del primo '\n' deve contenere solo ' ' e '\t'.
		txt := last.Text
		var lastCut = len(txt)
		for i := 0; i < len(txt); i++ {
			c := txt[i]
			if c == '\n' {
				lastCut = i + 1
				if i+1 < len(txt) && txt[i+1] == '\r' {
					lastCut++
				}
				break
			}
			if c != ' ' && c != '\t' {
				return
			}
		}
		last.Cut.Left = lastCut
	}
	if first != nil {
		first.Cut.Right = firstCut
	}
}
