// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scrigo

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	"scrigo/ast"
	"scrigo/parser"
)

// Error records a rendering error with the path and the position where
// the error occurred.
type Error struct {
	Path string
	Pos  ast.Position
	Err  error
}

func (err *Error) Error() string {
	return fmt.Sprintf("%s:%s: %s", err.Path, err.Pos, err.Err)
}

// errBreak returned from rendering the "break" statement.
// It is managed by the innermost "for" statement.
var errBreak = errors.New("break is not in a loop or switch")

// errContinue is returned from the rendering of the "break" statement.
// It is managed by the innermost "for" statement.
var errContinue = errors.New("continue is not in a loop")

// A returnError is returned from the rendering of the "return" statement.
// It is managed by the innermost "func" statement.
type returnError struct {
	args []interface{}
}

func (ret returnError) Error() string {
	return "scrigo: not captured return"
}

// rendering represents the state of a tree rendering.
type rendering struct {
	scope          map[string]scope
	path           string
	vars           []scope
	packages       map[string]*packageNameScope
	treeContext    ast.Context
	function       function
	handleError    func(error) bool
	needsReference map[*ast.Identifier]bool
	packageInfos   map[string]*parser.PackageInfo
	isScript       bool
}

// variables scope.
type scope map[string]interface{}

// reference represents a value in scope that has been referenced.
// rv contains the value, not the address.
type reference struct {
	rv reflect.Value
}

var scopeType = reflect.TypeOf(scope{})

// macro represents a macro in a scope.
type macro struct {
	path string
	node *ast.Macro
}

// function represents a function in a scope.
type function struct {
	path   string
	node   *ast.Func
	vars   []scope
	goFunc interface{}
}

func newFunction(path string, node *ast.Func, vars []scope) function {
	return function{path: path, node: node, vars: vars}
}

func (fn function) Func() interface{} {
	if fn.goFunc == nil {
		// TODO: compattare gli scopes
		fn.goFunc = reflect.MakeFunc(fn.node.Type.Reflect, func(args []reflect.Value) []reflect.Value {
			arguments := scope{}
			for i, p := range fn.node.Type.Parameters {
				if name := p.Ident.Name; name != "_" {
					arguments[name] = args[i].Interface()
				}
			}
			result, err := fn.call(arguments)
			if err != nil {
				panic(err)
			}
			return result
		}).Interface()
	}
	return fn.goFunc
}

func (fn function) call(args scope) ([]reflect.Value, error) {

	vars := make([]scope, len(fn.vars)+1)
	copy(vars, fn.vars)
	vars = append(vars, args)

	rn := &rendering{
		path:        fn.path,
		vars:        vars,
		treeContext: ast.ContextNone,
		handleError: stopOnError,
		function:    fn,
	}
	err := rn.render(nil, fn.node.Body.Nodes, nil)
	ret, ok := err.(returnError)
	if !ok {
		return nil, err
	}

	result := make([]reflect.Value, len(ret.args))
	for i, value := range ret.args {
		result[i] = reflect.ValueOf(value)
	}

	return result, nil
}

// urlState represents the rendering of rendering an URL.
type urlState struct {
	path   bool
	query  bool
	isSet  bool
	addAmp bool
}

// errorf builds and returns an rendering error.
func (r *rendering) errorf(nodeOrPos interface{}, format string, args ...interface{}) error {
	var pos *ast.Position
	if node, ok := nodeOrPos.(ast.Node); ok {
		pos = node.Pos()
		if pos == nil {
			return fmt.Errorf(format, args...)
		}
	} else {
		pos = nodeOrPos.(*ast.Position)
	}
	var err = &Error{
		Path: r.path,
		Pos: ast.Position{
			Line:   pos.Line,
			Column: pos.Column,
			Start:  pos.Start,
			End:    pos.End,
		},
		Err: fmt.Errorf(format, args...),
	}
	return err
}

func (r *rendering) renderBlock(wr io.Writer, block *ast.Block, urlstate *urlState) error {
	var err error
	r.vars = append(r.vars, nil)
	err = r.render(wr, block.Nodes, urlstate)
	r.vars = r.vars[:len(r.vars)-1]
	return err
}

// render renders nodes.
func (r *rendering) render(wr io.Writer, nodes []ast.Node, urlstate *urlState) error {

Nodes:
	for i, n := range nodes {

		switch node := n.(type) {

		case *ast.Text:

			if wr != nil {
				if len(node.Text)-node.Cut.Left-node.Cut.Right == 0 {
					continue
				}
				text := node.Text[node.Cut.Left : len(node.Text)-node.Cut.Right]
				if urlstate != nil {
					if !urlstate.query {
						if bytes.ContainsAny(text, "?#") {
							if text[0] == '?' && !urlstate.path {
								if urlstate.addAmp {
									_, err := io.WriteString(wr, "&amp;")
									if err != nil {
										return err
									}
								}
								text = text[1:]
							}
							urlstate.path = false
							urlstate.query = true
						}
						if urlstate.isSet && bytes.IndexByte(text, ',') >= 0 {
							urlstate.path = true
							urlstate.query = false
						}
					}
				}
				_, err := wr.Write(text)
				if err != nil {
					return err
				}
			}

		case *ast.URL:

			if len(node.Value) > 0 {
				isSet := node.Attribute == "srcset"
				err := r.render(wr, node.Value, &urlState{true, false, isSet, false})
				if err != nil {
					return err
				}
			}

		case *ast.Show:

			expr, err := r.eval(node.Expr)
			if err != nil {
				if r.handleError(err) {
					continue
				}
				return err
			}

			err = r.renderValue(wr, expr, node, urlstate)
			if err != nil {
				return err
			}

		case *ast.If:

			var c bool
			var err error

			r.vars = append(r.vars, nil)

			if node.Assignment != nil {
				err = r.renderAssignment(node.Assignment)
				if err != nil && !r.handleError(err) {
					return err
				}
			}
			if err == nil {
				expr, err := r.eval(node.Condition)
				if err != nil {
					if !r.handleError(err) {
						return err
					}
					expr = false
				}
				c = expr.(bool)
			}
			if c {
				if node.Then != nil && len(node.Then.Nodes) > 0 {
					err = r.renderBlock(wr, node.Then, urlstate)
					if err != nil {
						return err
					}
				}
			} else if node.Else != nil {
				var err error
				switch e := node.Else.(type) {
				case *ast.If:
					err = r.render(wr, []ast.Node{e}, urlstate)
				case *ast.Block:
					err = r.renderBlock(wr, e, urlstate)
				}
				if err != nil {
					return err
				}
			}

			r.vars = r.vars[:len(r.vars)-1]

		case *ast.For, *ast.ForRange:

			err := r.renderFor(wr, node, urlstate)
			if err != nil {
				return err
			}

		case *ast.Switch:

			// TODO (Gianluca): handle errors correctly using handleError if
			// necessary.

			var err error
			r.vars = append(r.vars, nil)

			if node.Init != nil {
				switch v := node.Init.(type) {
				case ast.Expression:
					_, err = r.eval(v)
					if err != nil && !r.handleError(err) {
						return err
					}
				case *ast.Assignment:
					err = r.renderAssignment(v)
					if err != nil && !r.handleError(err) {
						return err
					}
				}
			}

			r.vars = append(r.vars, nil)

			var conditionValue interface{}
			if node.Expr == nil {
				conditionValue = true
			} else {
				conditionValue, err = r.eval(node.Expr)
				if err != nil && !r.handleError(err) {
					return err
				}
			}

			inFallthrough := false
			var defaultCase *ast.Case
			for _, c := range node.Cases {
				// TODO (Gianluca): init render as true and invert if condition
				// removing else (if possibile)
				render := false
				isDefault := c.Expressions == nil
				if isDefault {
					defaultCase = c
				}
				if inFallthrough {
					render = true
				} else {
					for _, expr := range c.Expressions {
						res, err := r.evalBinary(conditionValue, ast.OperatorEqual, node.Expr, expr)
						if err != nil && !r.handleError(err) {
							return err
						}
						if render = res.(bool); render {
							break
						}
					}
				}
				if render {
					if inFallthrough {
						r.vars = r.vars[:len(r.vars)-1]
						r.vars = append(r.vars, nil)
					}
					err := r.render(wr, c.Body, urlstate)
					if err != nil {
						if err == errBreak {
							break
						}
						return err
					}
					if c.Fallthrough {
						inFallthrough = true
						continue
					}
					defaultCase = nil
					break
				}
			}

			if defaultCase != nil {
				err := r.render(wr, defaultCase.Body, urlstate)
				if err != nil {
					return err
				}
			}

			r.vars = r.vars[:len(r.vars)-2]

		case *ast.TypeSwitch:
			// TODO (Gianluca): apply the same changes requested for case
			// *ast.Switch.

			var err error

			r.vars = append(r.vars, nil)

			if node.Init != nil {
				switch v := node.Init.(type) {
				case ast.Expression:
					_, err = r.eval(v)
					if err != nil && !r.handleError(err) {
						return err
					}
				case *ast.Assignment:
					err = r.renderAssignment(v)
					if err != nil && !r.handleError(err) {
						return err
					}
				}
			}

			ident := node.Assignment.Variables[0].(*ast.Identifier)
			expr := node.Assignment.Values[0]

			r.vars = append(r.vars, nil)

			var guardvalue interface{}

			// An assignment with a blank identifier is a type
			// assertion encapsulated within an assignment.
			if ident.Name == "_" {
				expr = node.Assignment.Values[0]
				guardvalue, err = r.eval(expr)
				if err != nil {
					return err
				}
			} else {
				err = r.renderAssignment(node.Assignment)
				if err != nil && !r.handleError(err) {
					return err
				}
				guardvalue, _ = r.variable(ident.Name)
			}
			var defaultCase *ast.Case
			for _, c := range node.Cases {
				render := false
				isDefault := c.Expressions == nil
				if isDefault {
					defaultCase = c
				} else {
					for _, expr := range c.Expressions {
						if !isDefault {
							caseExprValue, err := r.eval(expr)
							if err != nil && !r.handleError(err) {
								return err
							}
							vt2, ok := caseExprValue.(reflect.Type)
							if !ok {
								return r.errorf(c, "%s (type %s) is not a type", expr, typeof(caseExprValue))
							}
							render, err = hasType(guardvalue, vt2)
							if err != nil {
								// TODO: use r.errorf
								return err
							}
						}
						if render {
							break
						}
					}
				}
				if render {
					err := r.render(wr, c.Body, urlstate)
					if err != nil {
						if err == errBreak {
							break
						}
						return err
					}
					defaultCase = nil
					break
				}
			}

			if defaultCase != nil {
				err := r.render(wr, defaultCase.Body, urlstate)
				if err != nil {
					return err
				}
			}

			r.vars = r.vars[:len(r.vars)-2]

		case *ast.Break:
			return errBreak

		case *ast.Continue:
			return errContinue

		case *ast.Block:
			err := r.renderBlock(wr, node, urlstate)
			if err != nil {
				return err
			}

		case *ast.Func:
			if wr != nil {
				err := r.errorf(node.Ident, "functions not allowed")
				if !r.handleError(err) {
					return err
				}
			}
			name := node.Ident.Name
			if name == "_" {
				continue
			}
			r.vars[2][name] = newFunction(r.path, node, r.vars[0:3])
			isLastScriptStatement := node.Ident == nil
			if isLastScriptStatement {
				fn := newFunction(r.path, node, r.vars[0:2])
				return returnError{args: []interface{}{fn.Func()}}
			}

		case *ast.Return:
			if wr != nil {
				err := r.errorf(node, "return not allowed")
				if !r.handleError(err) {
					return err
				}
			}
			result := r.function.node.Type.Result
			size := len(result)
			if size == 0 {
				return nil
			}
			var err error
			types := make([]reflect.Type, size)
			for i := size - 1; i >= 0; i-- {
				if t := result[i].Type; t == nil {
					types[i] = types[i+1]
				} else {
					types[i], err = r.evalType(t, noEllipses)
					if err != nil {
						return err
					}
				}
			}
			ret := returnError{args: make([]interface{}, size)}
			for i, value := range node.Values {
				v, err := r.eval1(value)
				if err != nil {
					return err
				}
				v, err = asAssignableTo(v, types[i])
				if err != nil {
					return err
				}
				ret.args[i] = v
			}
			return ret

		case *ast.Macro:
			if wr != nil {
				err := r.errorf(node.Ident, "macros not allowed")
				if !r.handleError(err) {
					return err
				}
			}
			name := node.Ident.Name
			if name == "_" {
				continue
			}
			if v, ok := r.variable(name); ok {
				var err error
				if m, ok := v.(macro); ok {
					err = r.errorf(node, "%s redeclared\n\tprevious declaration at %s:%s",
						name, m.path, m.node.Pos())
				} else {
					err = r.errorf(node.Ident, "%s redeclared in this file", name)
				}
				if r.handleError(err) {
					continue
				}
				return err
			}
			r.vars[2][name] = macro{r.path, node}

		case *ast.Var:

			// TODO (Gianluca): this implementation is partial and incorrect,
			// review after implementation of type checker.

			if node.Type == nil {
				// var a, b = 3, 4
				variables := make([]ast.Expression, len(node.Identifiers))
				for i, ident := range node.Identifiers {
					variables[i] = ident
				}
				assign := ast.NewAssignment(node.Pos(), variables, ast.AssignmentDeclaration, node.Values)
				err := r.renderAssignment(assign)
				if err != nil && !r.handleError(err) {
					return err
				}
			} else {
				if node.Values == nil {
					// var a, b int
					// TODO (Gianluca): assign zero.
					panic("not implemented")

				} else {
					// var a, b int = 3, 4
					variables := make([]ast.Expression, len(node.Identifiers))
					for i, ident := range node.Identifiers {
						variables[i] = ident
					}
					assign := ast.NewAssignment(node.Pos(), variables, ast.AssignmentDeclaration, node.Values)
					err := r.renderAssignment(assign)
					if err != nil && !r.handleError(err) {
						return err
					}
				}
			}

		case *ast.Const:

			// TODO (Gianluca): this implementation is partial and incorrect,
			// review after implementation of type checker.

			if node.Type == nil {
				// const a, b = 3, 4
				variables := make([]ast.Expression, len(node.Identifiers))
				for i, ident := range node.Identifiers {
					variables[i] = ident
				}
				assign := ast.NewAssignment(node.Pos(), variables, ast.AssignmentDeclaration, node.Values)
				err := r.renderAssignment(assign)
				if err != nil && !r.handleError(err) {
					return err
				}
			} else {
				// var a, b int = 3, 4

				// TODO (Gianluca): this implementation is partial and
				// incorrect, review after implementation of type checker.
				variables := make([]ast.Expression, len(node.Identifiers))
				for i, ident := range node.Identifiers {
					variables[i] = ident
				}
				assign := ast.NewAssignment(node.Pos(), variables, ast.AssignmentDeclaration, node.Values)
				err := r.renderAssignment(assign)
				if err != nil && !r.handleError(err) {
					return err
				}
			}

		case *ast.Assignment:

			err := r.renderAssignment(node)
			if err != nil && !r.handleError(err) {
				return err
			}

		case *ast.ShowMacro:

			name := node.Macro.Name
			if node.Import != nil {
				name = node.Import.Name + "." + name
			}
			var m macro
			var err error
			if v, ok := r.variable(name); ok {
				if m, ok = v.(macro); ok {
					if node.Context != m.node.Context {
						err = r.errorf(node, "macro %s is defined in a different context (%s)",
							name, m.node.Context)
					}
				} else {
					err = r.errorf(node, "cannot show non-macro %s (type %s)", name, typeof(v))
				}
			} else {
				err = r.errorf(node, "macro %s not declared", name)
			}
			if err != nil {
				if r.handleError(err) {
					continue
				}
				return err
			}

			isVariadic := m.node.IsVariadic

			haveSize := len(node.Arguments)
			wantSize := len(m.node.Parameters)

			var last = wantSize - 1
			var arguments = scope{}
			var parameters = m.node.Parameters
			if isVariadic {
				if length := haveSize - wantSize + 1; length == 0 {
					arguments[parameters[last].Name] = []interface{}(nil)
				} else {
					arguments[parameters[last].Name] = make([]interface{}, length)
				}
			}
			for i, argument := range node.Arguments {
				arg, err := r.eval(argument)
				if err != nil {
					if r.handleError(err) {
						continue Nodes
					}
					return err
				}
				if isVariadic && i >= last {
					arguments[parameters[last].Name].([]interface{})[i-wantSize+1] = arg
				} else {
					arguments[parameters[i].Name] = arg
				}
			}
			rn := &rendering{
				scope:       r.scope,
				path:        m.path,
				vars:        []scope{r.vars[0], r.vars[1], r.scope[m.path], arguments},
				treeContext: r.treeContext,
				handleError: r.handleError,
			}
			err = rn.render(wr, m.node.Body, nil)
			if err != nil {
				return err
			}

		case *ast.Import:

			if r.treeContext == ast.ContextNone {
				pkg, ok := r.packages[node.Path]
				if !ok {
					packageNode := node.Tree.Nodes[0].(*ast.Package)
					sc, err := renderPackageBlock(packageNode, r.packageInfos, r.packages, node.Tree.Path)
					if err != nil {
						return err
					}
					pkg = &packageNameScope{name: packageNode.Name, scope: sc}
					r.packages[node.Path] = pkg
				}
				if node.Ident == nil {
					r.vars[2][pkg.name] = pkg
				} else {
					switch node.Ident.Name {
					case "_":
					case ".":
						for ident, v := range pkg.scope {
							r.vars[2][ident] = v
						}
					default:
						r.vars[2][node.Ident.Name] = pkg
					}
				}
			} else {
				if _, ok := r.scope[node.Tree.Path]; !ok {
					rn := &rendering{
						scope:       r.scope,
						path:        node.Tree.Path,
						vars:        []scope{r.vars[0], r.vars[1], {}},
						packages:    r.packages,
						treeContext: r.treeContext,
						handleError: r.handleError,
					}
					err := rn.render(nil, node.Tree.Nodes, nil)
					if err != nil {
						return err
					}
					r.scope[node.Tree.Path] = rn.vars[2]
					if node.Ident != nil && node.Ident.Name == "_" {
						continue
					}
					for name, m := range rn.vars[2] {
						if _, ok := m.(macro); ok {
							if strings.Index(name, ".") > 0 {
								continue
							}
							if fc, _ := utf8.DecodeRuneInString(name); !unicode.Is(unicode.Lu, fc) {
								continue
							}
							r.vars[2][name] = m
						}
					}
				}
			}

		case *ast.Include:

			r.vars = append(r.vars, nil)
			rn := &rendering{
				scope:       r.scope,
				path:        node.Tree.Path,
				vars:        r.vars,
				treeContext: r.treeContext,
				handleError: r.handleError,
			}
			err := rn.render(wr, node.Tree.Nodes, nil)
			r.vars = r.vars[:len(r.vars)-1]
			if err != nil {
				return err
			}

		case ast.Expression:

			var err error
			var res interface{}
			if call, ok := node.(*ast.Call); ok {
				res, err = r.evalCallN(call)
			} else {
				res, err = r.eval1(node)
			}
			if err != nil && !r.handleError(err) {
				return err
			}
			isLastScriptStatement := r.isScript && r.function.node == nil && i == len(nodes)-1
			if isLastScriptStatement {
				return returnError{args: []interface{}{res}}
			}

		}
	}

	return nil
}

// variable returns the value of the variable name in rendering r.
func (r *rendering) variable(name string) (interface{}, bool) {
	for i := len(r.vars) - 1; i >= 0; i-- {
		if r.vars[i] != nil {
			if v, ok := r.vars[i][name]; ok {
				switch v := v.(type) {
				case reference:
					return v.rv.Interface(), true
				default:
					return v, true
				}

			}
		}
	}
	return nil, false
}

// typeof returns the string representation of the type of v.
// If v is nil returns "nil".
// TODO (Gianluca): to review.
func typeof(v interface{}) string {
	switch vv := v.(type) {
	case nil:
		return "nil"
	case HTML:
		return "string"
	case ConstantNumber:
		return "untyped number"
	case CustomNumber:
		return vv.Name()
	}
	return fmt.Sprintf("%T", v)
}
