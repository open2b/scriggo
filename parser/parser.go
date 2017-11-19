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
	"unicode"
	"unicode/utf8"

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

	// indica se è stato esteso
	var isExtended = false

	// indica  se ci trova in una region
	var isInRegion = false

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
				var comma token
				switch tok.typ {
				case tokenComma:
					// syntax: for index, ident in expr
					comma = tok
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
					// syntax: for index in expr..expr
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
				var expr2 ast.Expression
				if tok.typ == tokenRange {
					// syntax: for index in expr..expr
					if ident != nil {
						return nil, &Error{"", *comma.pos, fmt.Errorf("unexpected %s, expecting \"in\"", comma)}
					}
					expr2, tok, err = parseExpr(lex)
					if err != nil {
						return nil, err
					}
					if expr == nil {
						return nil, &Error{"", *tok.pos, fmt.Errorf("expecting expression")}
					}
				} else if ident == nil {
					// syntax: for ident in expr
					ident = index
					index = nil
				}
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				pos.End = tok.pos.End
				node = ast.NewFor(pos, index, ident, expr, expr2, nil)
				addChild(parent, node)
				ancestors = append(ancestors, node)
				cutSpacesToken = true

			// break
			case tokenBreak:
				var loop bool
				for i := len(ancestors) - 1; i > 0; i-- {
					if _, loop = ancestors[i].(*ast.For); loop {
						break
					}
				}
				if !loop {
					return nil, &Error{"", *tok.pos, fmt.Errorf("break is not in a loop")}
				}
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				pos.End = tok.pos.End
				node = ast.NewBreak(pos)
				addChild(parent, node)
				cutSpacesToken = true

			// continue
			case tokenContinue:
				var loop bool
				for i := len(ancestors) - 1; i > 0; i-- {
					if _, loop = ancestors[i].(*ast.For); loop {
						break
					}
				}
				if !loop {
					return nil, &Error{"", *tok.pos, fmt.Errorf("continue is not in a loop")}
				}
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				pos.End = tok.pos.End
				node = ast.NewContinue(pos)
				addChild(parent, node)
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
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end, for, if or show", tok)}
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
				if isExtended && !isInRegion {
					return nil, &Error{"", *tok.pos, fmt.Errorf("show statement outside region")}
				}
				// region or path
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ == tokenIdentifier {
					// show <region>
					region := ast.NewIdentifier(tok.pos, string(tok.txt))
					tok, ok = <-lex.tokens
					if !ok {
						return nil, lex.err
					}
					// import
					var impor *ast.Identifier
					if tok.typ == tokenPeriod {
						tok, ok = <-lex.tokens
						if !ok {
							return nil, lex.err
						}
						if tok.typ != tokenIdentifier {
							return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting identifier", tok)}
						}
						impor = region
						region = ast.NewIdentifier(tok.pos, string(tok.txt))
						if fc, _ := utf8.DecodeRuneInString(region.Name); !unicode.Is(unicode.Lu, fc) {
							return nil, &Error{"", *tok.pos, fmt.Errorf("cannot refer to unexported region %s", region.Name)}
						}
						tok, ok = <-lex.tokens
						if !ok {
							return nil, lex.err
						}
					}
					var arguments []ast.Expression
					if tok.typ == tokenLeftParenthesis {
						// arguments
						arguments = []ast.Expression{}
						for {
							expr, tok, err = parseExpr(lex)
							if err != nil {
								return nil, err
							}
							if expr == nil {
								return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting expression", tok)}
							}
							arguments = append(arguments, expr)
							if tok.typ == tokenRightParenthesis {
								break
							}
							if tok.typ != tokenComma {
								return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting , or )", tok)}
							}
						}
						tok, ok = <-lex.tokens
						if !ok {
							return nil, lex.err
						}
						if tok.typ != tokenEndStatement {
							return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
						}
					}
					if tok.typ != tokenEndStatement {
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting ( or %%}", tok)}
					}
					pos.End = tok.pos.End
					node = ast.NewShowRegion(pos, impor, region, arguments, parserContext(ctx))
				} else if tok.typ == tokenInterpretedString || tok.typ == tokenRawString {
					// show <path>
					var path = unquoteString(tok.txt)
					if !isValidFilePath(path) {
						return nil, fmt.Errorf("invalid path %q at %s", path, tok.pos)
					}
					tok, ok = <-lex.tokens
					if !ok {
						return nil, lex.err
					}
					if tok.typ != tokenEndStatement {
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting ( or %%}", tok)}
					}
					pos.End = tok.pos.End
					node = ast.NewShowPath(pos, path, parserContext(ctx))
				} else {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting identifier o string", tok)}
				}
				addChild(parent, node)
				cutSpacesToken = true

			// extend
			case tokenExtend:
				if isExtended {
					return nil, &Error{"", *tok.pos, fmt.Errorf("extend already exists")}
				}
				if len(tree.Nodes) > 0 {
					if _, ok := tree.Nodes[0].(*ast.Text); !ok || len(tree.Nodes) > 1 {
						return nil, &Error{"", *tok.pos, fmt.Errorf("extend can only be the first statement")}
					}
				}
				var tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenInterpretedString && tok.typ != tokenRawString {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting string", tok)}
				}
				var path = unquoteString(tok.txt)
				if !isValidFilePath(path) {
					return nil, &Error{"", *tok.pos, fmt.Errorf("invalid extend path %q", path)}
				}
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				pos.End = tok.pos.End
				node = ast.NewExtend(pos, path)
				addChild(parent, node)
				isExtended = true

			// import
			case tokenImport:
				for i := len(ancestors) - 1; i > 0; i-- {
					switch ancestors[i].(type) {
					case *ast.For:
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end for", tok)}
					case *ast.If:
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end if", tok)}
					case *ast.Region:
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end region", tok)}
					}
				}
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				var ident *ast.Identifier
				if tok.typ == tokenIdentifier {
					ident = ast.NewIdentifier(tok.pos, string(tok.txt))
					tok, ok = <-lex.tokens
					if !ok {
						return nil, lex.err
					}
				}
				if tok.typ != tokenInterpretedString && tok.typ != tokenRawString {
					return nil, fmt.Errorf("unexpected %s, expecting string at %s", tok, tok.pos)
				}
				var path = unquoteString(tok.txt)
				if !isValidFilePath(path) {
					return nil, fmt.Errorf("invalid import path %q at %s", path, tok.pos)
				}
				if ident == nil {
					name, ok := ast.GetImportName(path)
					if !ok {
						return nil, fmt.Errorf("%q cannot be used as import ident at %s", name, tok.pos)
					}
				}
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
				}
				pos.End = tok.pos.End
				node = ast.NewImport(pos, ident, path)
				addChild(parent, node)
				cutSpacesToken = true

			// region
			case tokenRegion:
				for i := len(ancestors) - 1; i > 0; i-- {
					switch ancestors[i].(type) {
					case *ast.For:
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end for", tok)}
					case *ast.If:
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end if", tok)}
					case *ast.Region:
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting end region", tok)}
					}
				}
				// ident
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				if tok.typ != tokenIdentifier {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting identifier", tok)}
				}
				ident := ast.NewIdentifier(tok.pos, string(tok.txt))
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				var parameters []*ast.Identifier
				if tok.typ == tokenLeftParenthesis {
					// parameters
					parameters = []*ast.Identifier{}
					for {
						tok, ok = <-lex.tokens
						if !ok {
							return nil, lex.err
						}
						if tok.typ != tokenIdentifier {
							return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting identifier", tok)}
						}
						parameters = append(parameters, ast.NewIdentifier(tok.pos, string(tok.txt)))
						tok, ok = <-lex.tokens
						if !ok {
							return nil, lex.err
						}
						if tok.typ == tokenRightParenthesis {
							break
						}
						if tok.typ != tokenComma {
							return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting , or )", tok)}
						}
					}
					tok, ok = <-lex.tokens
					if !ok {
						return nil, lex.err
					}
					if tok.typ != tokenEndStatement {
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
					}
				} else if tok.typ != tokenEndStatement {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting ( or %%}", tok)}
				}
				pos.End = tok.pos.End
				node = ast.NewRegion(pos, ident, parameters, nil)
				addChild(parent, node)
				ancestors = append(ancestors, node)
				cutSpacesToken = true
				isInRegion = true

			// end
			case tokenEnd:
				if len(ancestors) == 1 {
					return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s", tok)}
				}
				tok, ok = <-lex.tokens
				if !ok {
					return nil, lex.err
				}
				parent := ancestors[len(ancestors)-1]
				if tok.typ != tokenEndStatement {
					switch parent.(type) {
					case *ast.For:
						if tok.typ != tokenFor {
							return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting for or %%}", tok)}
						}
					case *ast.If:
						if tok.typ != tokenIf {
							return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting if or %%}", tok)}
						}
					case *ast.Region:
						if tok.typ != tokenRegion {
							return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting region or %%}", tok)}
						}
					}
					tok, ok = <-lex.tokens
					if !ok {
						return nil, lex.err
					}
					if tok.typ != tokenEndStatement {
						return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting %%}", tok)}
					}
				}
				if _, ok := parent.(*ast.Region); ok {
					isInRegion = false
				}
				ancestors = ancestors[:len(ancestors)-1]
				cutSpacesToken = true

			default:
				return nil, &Error{"", *tok.pos, fmt.Errorf("unexpected %s, expecting for, if, show, extend, region or end", tok)}

			}

		// {{ }}
		case tokenStartValue:
			if isExtended && !isInRegion {
				return nil, &Error{"", *tok.pos, fmt.Errorf("value statement outside region")}
			}
			tokensInLine++
			expr, tok2, err := parseExpr(lex)
			if err != nil {
				return nil, err
			}
			if expr == nil {
				return nil, &Error{"", *tok2.pos, fmt.Errorf("expecting expression")}
			}
			if tok2.typ != tokenEndValue {
				return nil, &Error{"", *tok2.pos, fmt.Errorf("unexpected %s, expecting }}", tok2)}
			}
			tok.pos.End = tok2.pos.End
			var node = ast.NewValue(tok.pos, expr, parserContext(tok.ctx))
			addChild(parent, node)

		// comment
		case tokenComment:
			tokensInLine++
			var node = ast.NewComment(tok.pos, string(tok.txt[2:len(tok.txt)-2]))
			addChild(parent, node)
			cutSpacesToken = true

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
// ShowPath se presenti.
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

	var dir = path[:strings.LastIndex(path, "/")+1]

	// espande i sotto alberi
	err = p.expand(tree.Nodes, dir, nil)
	if err != nil {
		if err2, ok := err.(*Error); ok && err2.Path == "" {
			err2.Path = path
		}
		return nil, err
	}

	// determina il nodo extend se presente
	var extend = getExtendNode(tree)

	if extend != nil && extend.Ref.Tree == nil {
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
			// legge le region exportate
			var regions []expandedRegion
			for _, node := range tree.Nodes {
				if region, ok := node.(*ast.Region); ok {
					if c, _ := utf8.DecodeRuneInString(region.Ident.Name); unicode.Is(unicode.Lu, c) {
						regions = append(regions, expandedRegion{nil, region})
					}
				}
			}
			// espande gli i sotto alberi
			var d = exPath[:strings.LastIndexByte(exPath, '/')+1]
			err = p.expand(tr.Nodes, d, regions)
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
		extend.Ref.Tree = tr
	}

	tree.IsExpanded = true

	return tree, nil
}

type expandedRegion struct {
	imp *ast.Import
	reg *ast.Region
}

// expand espande i nodi Import e ShowPath dell'albero tree
// chiamando read e anteponendo dir al path passato come argomento.
func (p *Parser) expand(nodes []ast.Node, dir string, regions []expandedRegion) error {

	for _, node := range nodes {

		switch n := node.(type) {

		case *ast.For:

			err := p.expand(n.Nodes, dir, regions)
			if err != nil {
				return err
			}

		case *ast.If:

			err := p.expand(n.Then, dir, regions)
			if err != nil {
				return err
			}
			err = p.expand(n.Else, dir, regions)
			if err != nil {
				return err
			}

		case *ast.Region:

			for _, r := range regions {
				if (r.imp == nil || r.imp.Ident == nil) && r.reg.Ident.Name == n.Ident.Name {
					return &Error{"", *(n.Pos()), fmt.Errorf("region %s already defined at %s", n.Ident.Name, r.imp.Pos())}
				}
			}
			regions = append(regions, expandedRegion{nil, n})

		case *ast.ShowRegion:

			for _, r := range regions {
				var found = n.Region.Name == r.reg.Ident.Name && (n.Import == nil && (r.imp == nil || r.imp.Ident == nil) ||
					n.Import != nil && r.imp != nil && r.imp.Ident != nil && n.Import.Name == r.imp.Ident.Name)
				if found {
					n.Ref.Import = r.imp
					n.Ref.Region = r.reg
					break
				}
			}
			if n.Ref.Region == nil {
				return &Error{"", *(n.Pos()), fmt.Errorf("region %s not declared", n.Region.Name)}
			}
			haveSize := len(n.Arguments)
			wantSize := len(n.Ref.Region.Parameters)
			if haveSize != wantSize {
				have := "("
				for i := 0; i < haveSize; i++ {
					if i > 0 {
						have += ","
					}
					if i < wantSize {
						have += n.Ref.Region.Parameters[i].Name
					} else {
						have += "?"
					}
				}
				have += ")"
				want := "("
				for i, p := range n.Ref.Region.Parameters {
					if i > 0 {
						want += ","
					}
					want += p.Name
				}
				want += ")"
				name := n.Region.Name
				if n.Import != nil {
					name = n.Import.Name + " " + name
				}
				if haveSize < wantSize {
					return &Error{"", *(n.Pos()), fmt.Errorf("not enough arguments in show of %s\n\thave %s\n\twant %s", name, have, want)}
				}
				return &Error{"", *(n.Pos()), fmt.Errorf("too many arguments in show of %s\n\thave %s\n\twant %s", name, have, want)}
			}

		case *ast.Import:

			if n.Ident != nil {
				for _, r := range regions {
					if r.imp != nil && r.imp.Ident != nil && r.imp.Ident.Name == n.Ident.Name {
						return &Error{"", *(n.Pos()), fmt.Errorf("%s redeclared in this file", n.Ident.Name)}
					}
				}
			}
			var err error
			n.Ref.Tree, err = p.expandSubTree(dir, n.Path)
			if err != nil {
				if err == ErrNotExist {
					err = &Error{"", *(n.Pos()), fmt.Errorf("import path %q does not exist", n.Path)}
				}
				return err
			}
			exRegions := map[string]*ast.Region{}
			for _, node := range n.Ref.Tree.Nodes {
				if region, ok := node.(*ast.Region); ok {
					name := region.Ident.Name
					if c, _ := utf8.DecodeRuneInString(region.Ident.Name); unicode.Is(unicode.Lu, c) {
						exRegions[name] = region
					}
				}
			}
			if n.Ident == nil {
				for _, r2 := range regions {
					if (r2.imp == nil || r2.imp.Ident == nil) && exRegions[r2.reg.Ident.Name] != nil {
						pos := r2.reg.Pos()
						if r2.imp != nil {
							pos = r2.imp.Pos()
						}
						err = &Error{"", *(n.Pos()), fmt.Errorf("region %s already defined at %s", r2.reg.Ident.Name, pos)}
					}
				}
			}
			for _, r := range exRegions {
				regions = append(regions, expandedRegion{n, r})
			}

		case *ast.ShowPath:

			var err error
			n.Ref.Tree, err = p.expandSubTree(dir, n.Path)
			if err != nil {
				if err == ErrNotExist {
					err = &Error{"", *(n.Pos()), fmt.Errorf("showed path %q does not exist", n.Path)}
				}
				return err
			}

		}

	}

	for _, node := range nodes {
		if region, ok := node.(*ast.Region); ok {
			err := p.expand(region.Body, dir, regions)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Parser) expandSubTree(dir, path string) (*ast.Tree, error) {
	var err error
	if path[0] != '/' {
		path, err = toAbsolutePath(dir, path)
		if err != nil {
			return nil, err
		}
	}
	var tree *ast.Tree
	tree, err = p.read(path)
	if err != nil {
		return nil, err
	}
	// se non è espanso lo espande
	tree.Lock()
	if !tree.IsExpanded {
		var d = path[:strings.LastIndexByte(path, '/')+1]
		err = p.expand(tree.Nodes, d, nil)
	}
	tree.Unlock()
	if err != nil {
		if err2, ok := err.(*Error); ok {
			if strings.HasSuffix(err2.Error(), "does not exist") {
				err2.Path = path
			}
		}
		return nil, err
	}
	return tree, nil
}

func addChild(parent ast.Node, node ast.Node) {
	switch n := parent.(type) {
	case *ast.Tree:
		n.Nodes = append(n.Nodes, node)
	case *ast.Region:
		n.Body = append(n.Body, node)
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
