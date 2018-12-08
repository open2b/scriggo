// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	"open2b/template/ast"
	"open2b/template/parser"
)

// Context indicates the type of source that has to be rendered and controls
// how to escape the values to render.
type Context int

const (
	ContextText   Context = Context(ast.ContextText)
	ContextHTML   Context = Context(ast.ContextHTML)
	ContextCSS    Context = Context(ast.ContextCSS)
	ContextScript Context = Context(ast.ContextScript)
)

// Errors is an error that can be returned after a rendering and contains all
// errors on expressions and statements execution that has occurred during the
// rendering of the template.
//
// If the argument strict of a renderer function or Render type constructor is
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
func NewDirRenderer(dir string, strict bool, ctx Context) *DirRenderer {
	var r = parser.DirReader(dir)
	return &DirRenderer{parser: parser.New(r), strict: strict, ctx: ast.Context(ctx)}
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
	return &MapRenderer{parser: parser.New(r), strict: strict, ctx: ast.Context(ctx)}
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
	ErrInvalidPath = errors.New("template: invalid path")

	// ErrNotExist is returned from a rendering function of method when the
	// path passed as argument does not exist.
	ErrNotExist = errors.New("template: path does not exist")
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

// RenderTree renders tree and writes the result to out. The variables in vars
// are defined as global variables. If strict is true, even errors on
// expressions and statements execution stop the rendering. See the type
// Errors for more details.
//
// If you have template sources instead or you do not want to deal directly
// with trees, use the function RenderSource or the Render method of a
// Renderer as DirRenderer and MapRenderer.
//
// RenderTree is safe for concurrent use.
func RenderTree(out io.Writer, tree *ast.Tree, vars interface{}, strict bool) error {

	if out == nil {
		return errors.New("template: w is nil")
	}
	if tree == nil {
		return errors.New("template: tree is nil")
	}

	globals, err := varsToScope(vars)
	if err != nil {
		return err
	}

	s := &rendering{
		scope:       map[string]scope{},
		path:        tree.Path,
		vars:        []scope{builtins, globals, {}},
		treeContext: tree.Context,
	}

	var errs []*Error
	if strict {
		s.handleError = stopOnError
	} else {
		s.handleError = func(err error) bool {
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
		err = s.render(out, tree.Nodes, nil)
	} else {
		if extends.Tree == nil {
			return errors.New("template: extends node is not expanded")
		}
		s.scope[s.path] = s.vars[2]
		err = s.render(nil, tree.Nodes, nil)
		if err != nil {
			return err
		}
		s.path = extends.Tree.Path
		vars := scope{}
		for name, v := range s.vars[2] {
			if r, ok := v.(macro); ok {
				fc, _ := utf8.DecodeRuneInString(name)
				if unicode.Is(unicode.Lu, fc) && !strings.Contains(name, ".") {
					vars[name] = r
				}
			}
		}
		s.vars = []scope{builtins, globals, vars}
		err = s.render(out, extends.Tree.Nodes, nil)
	}

	if err == nil && errs != nil {
		err = Errors(errs)
	}

	return err
}

// varsToScope converts variables into a scope.
func varsToScope(vars interface{}) (scope, error) {

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
				if tag, ok := field.Tag.Lookup("template"); ok {
					name = parseVarTag(tag)
					if name == "" {
						return nil, fmt.Errorf("template: invalid tag of field %q", field.Name)
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
			return varsToScope(reflect.Indirect(rv))
		}
	}

	return nil, errors.New("template: unsupported vars type")
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
