// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scrigo

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	"scrigo/ast"
	"scrigo/parser"
)

type pkgConstant struct {
	value interface{}
	typ   reflect.Type // nil for untyped constants.
}

// Constant returns a constant with given value and type. Can be used in Scrigo
// packages definition. typ is nil for untyped constants.
func Constant(value interface{}, typ reflect.Type) pkgConstant {
	return pkgConstant{value, typ}
}

// Context indicates the type of source that has to be rendered and controls
// how to escape the values to render.
type Context int

const (
	ContextNone   Context = Context(ast.ContextNone)
	ContextText   Context = Context(ast.ContextText)
	ContextHTML   Context = Context(ast.ContextHTML)
	ContextCSS    Context = Context(ast.ContextCSS)
	ContextScript Context = Context(ast.ContextScript)
)

// Errors is an error that can be returned after a rendering and contains all
// errors on expressions and statements execution that has occurred during the
// rendering of the scrigo.
//
// If the strict argument of a renderer function or Render type constructor is
// true, these types of errors stop the execution. If no other type of error
// has occurred, at the end, an Errors error is returned with all errors on
// expressions and statements execution that have occurred.
//
// Syntax errors, not-existing paths, write errors and invalid global
// variable definitions are all types of errors that always stop rendering.
//
// Operations on wrong types, out of bound on a slice, too few arguments in
// function calls are all types of errors that stop the execution only if
// strict is true.
type Errors []*Error

func (errs Errors) Error() string {
	var s string
	for _, err := range errs {
		if s != "" {
			s += "\n"
		}
		s += err.Error()
	}
	return s
}

// Renderer is the interface that is implemented by types that render template
// sources given a path.
type Renderer interface {
	Render(out io.Writer, path string, vars interface{}) error
}

// DirRenderer allows to render files located in a directory with the same
// context. Files are read and parsed only the first time that are rendered.
// Subsequents renderings are faster to execute.
type DirRenderer struct {
	parser *parser.Parser
	strict bool
	ctx    ast.Context
}

// NewDirRenderer returns a Dir that render files located in the directory dir
// in the context ctx. If strict is true, even errors on expressions and
// statements execution stop the rendering. See the type Errors for more
// details.
func NewDirRenderer(dir string, strict bool, ctx Context, typeCheck bool) *DirRenderer {
	var r = parser.DirReader(dir)
	return &DirRenderer{parser: parser.New(r, nil, typeCheck), strict: strict, ctx: ast.Context(ctx)}
}

// Render renders the template file with the specified path, relative to the
// template directory, and writes the result to out. The variables in vars are
// defined as global variables.
//
// Render is safe for concurrent use.
func (dr *DirRenderer) Render(out io.Writer, path string, vars interface{}) error {
	tree, err := dr.parser.Parse(path, dr.ctx)
	if err != nil {
		return convertError(err)
	}
	return RenderTree(out, tree, vars, dr.strict)
}

// MapRenderer allows to render sources as values of a map with the same
// context. Sources are parsed only the first time that are rendered.
// Subsequents renderings are faster to execute.
type MapRenderer struct {
	parser *parser.Parser
	strict bool
	ctx    ast.Context
}

// NewMapRenderer returns a Map that render sources as values of a map in the
// context ctx. If strict is true, even errors on expressions and statements
// execution stop the rendering. See the type Errors for more details.
func NewMapRenderer(sources map[string][]byte, strict bool, ctx Context) *MapRenderer {
	var r = parser.MapReader(sources)
	return &MapRenderer{parser: parser.New(r, nil, false), strict: strict, ctx: ast.Context(ctx)}
}

// Render renders the template source with the specified path and writes
// the result to out. The variables in vars are defined as global variables.
//
// Render is safe for concurrent use.
func (mr *MapRenderer) Render(out io.Writer, path string, vars interface{}) error {
	tree, err := mr.parser.Parse(path, mr.ctx)
	if err != nil {
		return convertError(err)
	}
	return RenderTree(out, tree, vars, mr.strict)
}

var (
	// ErrInvalidPath is returned from a rendering function of method when the
	// path argument is not valid.
	ErrInvalidPath = errors.New("scrigo: invalid path")

	// ErrNotExist is returned from a rendering function of method when the
	// path passed as argument does not exist.
	ErrNotExist = errors.New("scrigo: path does not exist")
)

// RenderSource renders the template source src, in context ctx, and writes
// the result to out. The variables in vars are defined as global variables.
// If strict is true, even errors on expressions and statements execution stop
// the rendering. See the type Errors for more details.
//
// Statements "extends", "import" and "include" cannot be used with
// RenderSource, use the function RenderTree or the method Render of a
// Renderer, as DirRenderer and MapRenderer, instead.
//
// RenderSource is safe for concurrent use.
func RenderSource(out io.Writer, src []byte, vars interface{}, strict bool, ctx Context) error {
	tree, err := parser.ParseSource(src, ast.Context(ctx))
	if err != nil {
		return convertError(err)
	}
	return RenderTree(out, tree, vars, strict)
}

func stopOnError(err error) bool {
	return false
}

// Run reads a Scrigo source from r and executes it, returning any parsing or
// execution errors it encounters.
func Run(r io.Reader) error {
	src, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}
	tree, err := parser.ParseSource(src, ast.ContextNone)
	if err != nil {
		return err
	}
	return RenderTree(nil, tree, nil, true)
}

// RenderTree renders tree and writes the result to out. The variables in vars
// are defined as global variables. If strict is true, even errors on
// expressions and statements execution stop the rendering. See the type
// Errors for more details.
//
// If you have a template source instead of a tree, use the function
// RenderSource or the Render method of a Renderer as DirRenderer and
// MapRenderer.
//
// RenderTree is safe for concurrent use.
func RenderTree(out io.Writer, tree *ast.Tree, globals interface{}, strict bool) error {

	if out == nil {
		return errors.New("scrigo: out is nil")
	}
	if tree == nil {
		return errors.New("scrigo: tree is nil")
	}

	globalScope, err := globalsToScope(globals)
	if err != nil {
		return err
	}

	r := &rendering{
		scope:       map[string]scope{},
		path:        tree.Path,
		vars:        []scope{templateBuiltins, globalScope, {}},
		treeContext: tree.Context,
	}

	var errs []*Error
	if strict {
		r.handleError = stopOnError
	} else {
		r.handleError = func(err error) bool {
			if e, ok := err.(*Error); ok {
				for _, ex := range errs {
					if e.Path == ex.Path && e.Pos.Line == ex.Pos.Line &&
						e.Pos.Column == ex.Pos.Column && e.Err.Error() == ex.Err.Error() {
						return true
					}
				}
				errs = append(errs, e)
				return true
			}
			return false
		}
	}

	extends := getExtendsNode(tree)
	if extends == nil {
		err = r.render(out, tree.Nodes, nil)
	} else {
		if extends.Tree == nil {
			return errors.New("scrigo: extends node is not expanded")
		}
		r.scope[r.path] = r.vars[2]
		err = r.render(nil, tree.Nodes, nil)
		if err != nil {
			return err
		}
		r.path = extends.Tree.Path
		vars := scope{}
		for name, v := range r.vars[2] {
			if r, ok := v.(macro); ok {
				fc, _ := utf8.DecodeRuneInString(name)
				if unicode.Is(unicode.Lu, fc) && !strings.Contains(name, ".") {
					vars[name] = r
				}
			}
		}
		r.vars = []scope{templateBuiltins, globalScope, vars}
		err = r.render(out, extends.Tree.Nodes, nil)
	}

	if err == nil && errs != nil {
		err = Errors(errs)
	}

	return err
}

// RunScriptTree runs the tree of a script.
//
// Run is safe for concurrent use.
func RunScriptTree(tree *ast.Tree, globals interface{}) error {

	if tree == nil {
		return errors.New("scrigo: tree is nil")
	}
	if tree.Context != ast.ContextNone {
		return errors.New("scrigo: tree context is not None")
	}

	globalScope, err := globalsToScope(globals)
	if err != nil {
		return err
	}

	r := &rendering{
		scope:       map[string]scope{},
		path:        tree.Path,
		vars:        []scope{builtins, globalScope, {}},
		treeContext: ast.ContextNone,
		handleError: stopOnError,
	}

	return r.render(nil, tree.Nodes, nil)
}

func renderPackageBlock(astPkg *ast.Package, pkgInfos map[string]*parser.PackageInfo, pkgs map[string]*Package, path string) (map[string]scope, error) {

	r := &rendering{
		handleError:  stopOnError,
		packages:     pkgs,
		path:         path,
		scope:        map[string]scope{},
		treeContext:  ast.ContextNone,
		vars:         []scope{builtins, {}, {}},
		packageInfos: pkgInfos,
	}

	// nodes contains a list of declarations ordered by initialization
	// priority.
	nodes := make([]ast.Node, 0, len(astPkg.Declarations))

	// Imports.
	for _, node := range astPkg.Declarations {
		if _, ok := node.(*ast.Import); ok {
			nodes = append(nodes, node)
		}
	}

	// inits contains the list of "init" functions.
	var inits []*ast.Func

	// Functions.
	for _, node := range astPkg.Declarations {
		if f, ok := node.(*ast.Func); ok {
			if f.Ident.Name == "init" {
				inits = append(inits, f)
				continue
			}
			nodes = append(nodes, node)
		}
	}

	// Global variables, following initialization order.
	for _, varName := range pkgInfos[path].VariableOrdering {
		for _, n := range astPkg.Declarations {
			if varNode, ok := n.(*ast.Var); ok {
				for i, ident := range varNode.Identifiers {
					if ident.Name == varName {
						if len(varNode.Identifiers) != len(varNode.Values) {
							nodes = append(nodes, varNode)
							break
						}
						newVarDecl := ast.NewVar(varNode.Pos(), []*ast.Identifier{ident}, nil, []ast.Expression{varNode.Values[i]})
						nodes = append(nodes, newVarDecl)
						break
					}
				}
			}
		}
	}

	err := r.render(nil, nodes, nil)
	if err != nil {
		return nil, err
	}

	// Calls init functions.
	for _, f := range inits {
		err := r.render(nil, f.Body.Nodes, nil)
		if err != nil {
			return nil, err
		}
	}
	r.scope[path] = r.vars[2]
	return r.scope, nil
}

// RunPackageTree runs the tree of a main package.
//
// RunPackageTree is safe for concurrent use.
func RunPackageTree(tree *ast.Tree, packages map[string]*Package, pkgInfos map[string]*parser.PackageInfo) error {

	if tree == nil {
		return errors.New("scrigo: tree is nil")
	}
	if tree.Context != ast.ContextNone {
		return errors.New("scrigo: tree context is not None")
	}

	if len(tree.Nodes) != 1 {
		return errors.New("scrigo: tree must contains only the main package")
	}
	pkg, ok := tree.Nodes[0].(*ast.Package)
	if !ok {
		return errors.New("scrigo: tree must contains only the main package")
	}
	if pkg.Name != "main" {
		return &Error{tree.Path, *(pkg.Pos()), errors.New("package name must be main")}
	}

	r := &rendering{
		scope:          map[string]scope{},
		path:           tree.Path,
		vars:           []scope{builtins, {}, {}},
		packages:       packages,
		treeContext:    ast.ContextNone,
		handleError:    stopOnError,
		needsReference: pkgInfos[tree.Path].UpValues,
		packageInfos:   pkgInfos,
	}

	var err error
	r.scope, err = renderPackageBlock(pkg, pkgInfos, packages, tree.Path)
	if err != nil {
		return err
	}
	r.vars[2] = r.scope[tree.Path]
	mf := r.vars[2]["main"]
	r.scope[tree.Path] = r.vars[2]
	r.vars = append(r.vars, scope{}) // adds 'main' function scope?
	r.function = mf.(function)
	err = r.render(nil, r.function.node.Body.Nodes, nil)
	if err != nil {
		return err
	}

	return nil
}

// globalsToScope converts globals into a scope.
func globalsToScope(vars interface{}) (scope, error) {

	if vars == nil {
		return scope{}, nil
	}

	var rv reflect.Value
	if rv, ok := vars.(reflect.Value); ok {
		vars = rv.Interface()
	}

	if v, ok := vars.(map[string]interface{}); ok {
		return scope(v), nil
	}

	if !rv.IsValid() {
		rv = reflect.ValueOf(vars)
	}
	rt := rv.Type()

	switch rv.Kind() {
	case reflect.Map:
		if rt.ConvertibleTo(scopeType) {
			m := rv.Convert(scopeType).Interface()
			return m.(scope), nil
		}
		if rt.Key().Kind() == reflect.String {
			s := scope{}
			for _, kv := range rv.MapKeys() {
				s[kv.String()] = rv.MapIndex(kv).Interface()
			}
			return s, nil
		}
	case reflect.Struct:
		type st struct {
			rt reflect.Type
			rv reflect.Value
		}
		globals := scope{}
		structs := []st{{rt, rv}}
		var s st
		for len(structs) > 0 {
			s, structs = structs[0], structs[1:]
			nf := s.rv.NumField()
			for i := 0; i < nf; i++ {
				field := s.rt.Field(i)
				if field.PkgPath != "" {
					continue
				}
				if field.Anonymous {
					switch field.Type.Kind() {
					case reflect.Ptr:
						elem := field.Type.Elem()
						if elem.Kind() == reflect.Struct {
							if ptr := s.rv.Field(i); !ptr.IsNil() {
								value := reflect.Indirect(ptr)
								structs = append(structs, st{elem, value})
							}
						}
					case reflect.Struct:
						structs = append(structs, st{field.Type, s.rv.Field(i)})
					}
					continue
				}
				value := s.rv.Field(i).Interface()
				var name string
				if tag, ok := field.Tag.Lookup("scrigo"); ok {
					name = parseVarTag(tag)
					if name == "" {
						return nil, fmt.Errorf("scrigo: invalid tag of field %q", field.Name)
					}
				}
				if name == "" {
					name = field.Name
				}
				if _, ok := globals[name]; !ok {
					globals[name] = value
				}
			}
		}
		return globals, nil
	case reflect.Ptr:
		elem := rv.Type().Elem()
		if elem.Kind() == reflect.Struct {
			return globalsToScope(reflect.Indirect(rv))
		}
	}

	return nil, errors.New("scrigo: unsupported globals type")
}

// getExtendsNode returns the Extends node of a tree.
// If the node is not present, returns nil.
func getExtendsNode(tree *ast.Tree) *ast.Extends {
	if len(tree.Nodes) == 0 {
		return nil
	}
	if node, ok := tree.Nodes[0].(*ast.Extends); ok {
		return node
	}
	if len(tree.Nodes) > 1 {
		if node, ok := tree.Nodes[1].(*ast.Extends); ok {
			return node
		}
	}
	return nil
}

func convertError(err error) error {
	if err == parser.ErrInvalidPath {
		return ErrInvalidPath
	}
	if err == parser.ErrNotExist {
		return ErrNotExist
	}
	return err
}

type Program struct {
	tree      *ast.Tree
	packages  map[string]*Package
	typecheck map[string]*parser.PackageInfo
}

type Script struct {
	tree      *ast.Tree
	typecheck map[string]*parser.PackageInfo
	main      *parser.GoPackage
}

type Template struct {
	reader parser.Reader
}

type Page struct {
	tree *ast.Tree
}

type PackageReader struct {
	src    io.Reader
	reader parser.Reader
}

func (pr PackageReader) Read(path string, _ ast.Context) ([]byte, error) {
	if path == "/main" {
		src, err := ioutil.ReadAll(pr.src)
		if err != nil {
			return nil, err
		}
		return src, nil
	}
	return pr.reader.Read(path, ast.ContextNone)
}

func NewTemplate(reader parser.Reader) *Template {
	return &Template{reader: reader}
}

func (t *Template) Compile(path string, main *parser.GoPackage, ctx Context) (*Page, error) {
	packages := map[string]*parser.GoPackage{"main": main}
	p := parser.New(t.reader, packages, false)
	tree, err := p.Parse(path, ast.Context(ctx))
	if err != nil {
		return nil, convertError(err)
	}
	return &Page{tree: tree}, nil
}

func Render(out io.Writer, page *Page, vars map[string]interface{}) error {
	return RenderTree(out, page.tree, vars, true)
}

type Compiler struct {
	reader   parser.Reader
	packages map[string]*parser.GoPackage
}

func NewCompiler(reader parser.Reader, packages map[string]*parser.GoPackage) *Compiler {
	return &Compiler{reader: reader, packages: packages}
}

func (c *Compiler) Compile(src io.Reader) (*Program, error) {
	var program = &Program{}
	pr := PackageReader{src: src, reader: c.reader}
	p := parser.New(pr, c.packages, true)
	tree, err := p.Parse("/main", ast.ContextNone)
	if err != nil {
		return nil, err
	}
	if _, ok := tree.Nodes[0].(*ast.Package); !ok {
		return nil, errors.New("expected package") // TODO (Gianluca): to review.
	}
	program.tree = tree
	program.typecheck = p.TypeCheckInfos()
	program.packages = make(map[string]*Package, len(c.packages))
	for n, pkg := range c.packages {
		program.packages[n] = &Package{Name: pkg.Name, Declarations: pkg.Declarations}
	}
	return program, nil
}

func CompileScript(src io.Reader, main *parser.GoPackage) (*Script, error) {
	buf, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, err
	}
	r := parser.MapReader{"/main": buf}
	p := parser.New(r, map[string]*parser.GoPackage{"main": main}, true)
	tree, err := p.Parse("/main", ast.ContextNone)
	if err != nil {
		return nil, err
	}
	if _, ok := tree.Nodes[0].(*ast.Package); ok {
		return nil, errors.New("unexpected package") // TODO (Gianluca): to review.
	}
	return &Script{tree: tree, typecheck: p.TypeCheckInfos(), main: main}, nil
}

func Execute(p *Program) error {
	return RunPackageTree(p.tree, p.packages, p.typecheck)
}

func ExecuteScript(s *Script, vars map[string]interface{}) error {
	mainValues, err := globalsToScope(s.main)
	if err != nil {
		return err
	}
	for n, v := range vars {
		mainValues[n] = v
	}
	r := &rendering{
		scope:       map[string]scope{},
		path:        s.tree.Path,
		vars:        []scope{builtins, mainValues, {}},
		treeContext: ast.ContextNone,
		handleError: stopOnError,
	}
	return r.render(nil, s.tree.Nodes, nil)
}
