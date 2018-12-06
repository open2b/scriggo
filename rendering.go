// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	"open2b/template/ast"

	"github.com/shopspring/decimal"
)

var maxInt = decimal.New(int64(^uint(0)>>1), 0)
var minInt = decimal.New(-int64(^uint(0)>>1)-1, 0)

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
var errBreak = errors.New("break is not in a loop")

// errContinue is returned from the rendering of the "break" statement.
// It is managed by the innermost "for" statement.
var errContinue = errors.New("continue is not in a loop")

// rendering represents the state of a tree rendering.
type rendering struct {
	scope       map[string]scope
	path        string
	vars        []scope
	treeContext ast.Context
	handleError func(error) bool
}

// variables scope.
type scope map[string]interface{}

var scopeType = reflect.TypeOf(scope{})

// macro represents a macro in a scope.
type macro struct {
	path string
	node *ast.Macro
}

// urlState represents the rendering of rendering an URL.
type urlState struct {
	path   bool
	query  bool
	isSet  bool
	addAmp bool
}

// errorf builds and returns an rendering error.
func (s *rendering) errorf(node ast.Node, format string, args ...interface{}) error {
	var pos = node.Pos()
	if pos == nil {
		return fmt.Errorf(format, args...)
	}
	var err = &Error{
		Path: s.path,
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

// render renders nodes.
func (s *rendering) render(wr io.Writer, nodes []ast.Node, urlstate *urlState) error {

Nodes:
	for _, n := range nodes {

		switch node := n.(type) {

		case *ast.Text:

			if wr != nil {
				if node.Cut.Left == node.Cut.Right {
					continue
				}
				text := node.Text[node.Cut.Left:node.Cut.Right]
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
				err := s.render(wr, node.Value, &urlState{true, false, isSet, false})
				if err != nil {
					return err
				}
			}

		case *ast.Value:

			expr, err := s.eval(node.Expr)
			if err != nil {
				if s.handleError(err) {
					continue
				}
				return err
			}

			err = s.renderValue(wr, expr, node, urlstate)
			if err != nil {
				return err
			}

		case *ast.If:

			expr, err := s.eval(node.Expr)
			if err != nil {
				if !s.handleError(err) {
					return err
				}
				expr = false
			}
			c, ok := expr.(bool)
			if !ok {
				err = s.errorf(node, "non-bool %s (type %s) used as if condition", node.Expr, typeof(expr))
				if !s.handleError(err) {
					return err
				}
			}
			if c {
				if len(node.Then) > 0 {
					s.vars = append(s.vars, nil)
					err = s.render(wr, node.Then, urlstate)
					s.vars = s.vars[:len(s.vars)-1]
					if err != nil {
						return err
					}
				}
			} else if len(node.Else) > 0 {
				s.vars = append(s.vars, nil)
				err = s.render(wr, node.Else, urlstate)
				s.vars = s.vars[:len(s.vars)-1]
				if err != nil {
					return err
				}
			}

		case *ast.For:

			index := ""
			if node.Index != nil {
				index = node.Index.Name
			}
			ident := ""
			if node.Ident != nil {
				ident = node.Ident.Name
			}

			expr, err := s.eval(node.Expr1)
			if err != nil {
				if s.handleError(err) {
					continue
				}
				return err
			}

			if len(node.Nodes) == 0 {
				continue
			}

			if node.Expr2 == nil {
				// Syntax: for ident in expr
				// Syntax: for index, ident in expr

				setScope := func(i int, v interface{}) {
					vars := scope{ident: v}
					if index != "" {
						vars[index] = i
					}
					s.vars[len(s.vars)-1] = vars
				}

				switch vv := expr.(type) {
				case string:
					size := len(vv)
					if size == 0 {
						continue
					}
					s.vars = append(s.vars, nil)
					i := 0
					for _, v := range vv {
						setScope(i, string(v))
						err = s.render(wr, node.Nodes, urlstate)
						if err != nil {
							if err == errBreak {
								break
							}
							if err != errContinue {
								return err
							}
						}
						i++
					}
					s.vars = s.vars[:len(s.vars)-1]
				case []string:
					size := len(vv)
					if size == 0 {
						continue
					}
					s.vars = append(s.vars, nil)
					for i := 0; i < size; i++ {
						setScope(i, vv[i])
						err = s.render(wr, node.Nodes, urlstate)
						if err != nil {
							if err == errBreak {
								break
							}
							if err != errContinue {
								return err
							}
						}
					}
					s.vars = s.vars[:len(s.vars)-1]
				default:
					av := reflect.ValueOf(expr)
					if !av.IsValid() || av.Kind() != reflect.Slice {
						err = s.errorf(node, "cannot range over %s (type %s)", node.Expr1, typeof(expr))
						if s.handleError(err) {
							continue
						}
						return err
					}
					size := av.Len()
					if size == 0 {
						continue
					}
					s.vars = append(s.vars, nil)
					for i := 0; i < size; i++ {
						setScope(i, av.Index(i).Interface())
						err = s.render(wr, node.Nodes, urlstate)
						if err != nil {
							if err == errBreak {
								break
							}
							if err != errContinue {
								return err
							}
						}
					}
					s.vars = s.vars[:len(s.vars)-1]
				}

			} else {
				// Syntax: for index in expr..expr

				expr2, err := s.eval(node.Expr2)
				if err != nil {
					if s.handleError(err) {
						continue
					}
					return err
				}

				var n1 int
				switch n := expr.(type) {
				case int:
					n1 = n
				case decimal.Decimal:
					n1, err = s.decimalToInt(node.Expr1, n)
				default:
					err = s.errorf(node, "non-integer for range %s", node.Expr1)
				}
				if err != nil {
					if s.handleError(err) {
						continue
					}
					return err
				}

				var n2 int
				switch n := expr2.(type) {
				case int:
					n2 = n
				case decimal.Decimal:
					n2, err = s.decimalToInt(node.Expr2, n)
				default:
					err = s.errorf(node, "non-integer for range %s", node.Expr2)
				}
				if err != nil {
					if s.handleError(err) {
						continue
					}
					return err
				}

				step := 1
				if n2 < n1 {
					step = -1
				}

				s.vars = append(s.vars, nil)
				for i := n1; ; i += step {
					s.vars[len(s.vars)-1] = scope{index: i}
					err = s.render(wr, node.Nodes, urlstate)
					if err != nil {
						if err == errBreak {
							break
						}
						if err == errContinue {
							continue
						}
						return err
					}
					if i == n2 {
						break
					}
				}
				s.vars = s.vars[:len(s.vars)-1]

			}

		case *ast.Break:
			return errBreak

		case *ast.Continue:
			return errContinue

		case *ast.Macro:
			if wr != nil {
				err := s.errorf(node.Ident, "macros not allowed")
				if !s.handleError(err) {
					return err
				}
			}
			name := node.Ident.Name
			if v, ok := s.variable(name); ok {
				var err error
				if m, ok := v.(macro); ok {
					err = s.errorf(node, "%s redeclared\n\tprevious declaration at %s:%s",
						name, m.path, m.node.Pos())
				} else {
					err = s.errorf(node.Ident, "%s redeclared in this file", name)
				}
				if s.handleError(err) {
					continue
				}
				return err
			}
			s.vars[2][name] = macro{s.path, node}

		case *ast.Var:

			if s.vars[len(s.vars)-1] == nil {
				s.vars[len(s.vars)-1] = scope{}
			}
			var vars = s.vars[len(s.vars)-1]
			var name = node.Ident.Name
			if v, ok := vars[name]; ok {
				var err error
				if m, ok := v.(macro); ok {
					err = s.errorf(node, "%s redeclared\n\tprevious declaration at %s:%s",
						name, m.path, m.node.Pos())
				} else {
					err = s.errorf(node.Ident, "%s redeclared in this block", name)
				}
				if s.handleError(err) {
					continue
				}
				return err
			}
			v, err := s.eval(node.Expr)
			if err != nil {
				if s.handleError(err) {
					continue
				}
				return err
			}
			vars[name] = v

		case *ast.Assignment:

			var found bool
			var name = node.Ident.Name
			for i := len(s.vars) - 1; i >= 0; i-- {
				vars := s.vars[i]
				if vars != nil {
					if v, ok := vars[name]; ok {
						var err error
						if _, ok := v.(macro); ok || i < 2 {
							if i == 0 && name == "len" {
								err = s.errorf(node, "use of builtin len not in function call")
							} else {
								err = s.errorf(node, "cannot assign to %s", name)
							}
							if s.handleError(err) {
								continue Nodes
							}
							return err
						}
						v, err = s.eval(node.Expr)
						if err != nil {
							if s.handleError(err) {
								continue Nodes
							}
							return err
						}
						vars[name] = v
						found = true
						break
					}
				}
			}
			if !found {
				err := s.errorf(node, "variable %s not declared", name)
				if !s.handleError(err) {
					return err
				}
			}

		case *ast.ShowMacro:

			name := node.Macro.Name
			if node.Import != nil {
				name = node.Import.Name + "." + name
			}
			var m macro
			var err error
			if v, ok := s.variable(name); ok {
				if m, ok = v.(macro); ok {
					if node.Context != m.node.Context {
						err = s.errorf(node, "macro %s is defined in a different context (%s)",
							name, m.node.Context)
					}
				} else {
					err = s.errorf(node, "cannot show non-macro %s (type %s)", name, typeof(v))
				}
			} else {
				err = s.errorf(node, "macro %s not declared", name)
			}
			if err != nil {
				if s.handleError(err) {
					continue
				}
				return err
			}

			isVariadic := m.node.IsVariadic

			haveSize := len(node.Arguments)
			wantSize := len(m.node.Parameters)
			if (!isVariadic && haveSize != wantSize) || (isVariadic && haveSize < wantSize-1) {
				have := "("
				for i := 0; i < haveSize; i++ {
					if i > 0 {
						have += ", "
					}
					if i < wantSize {
						have += m.node.Parameters[i].Name
					} else {
						have += "?"
					}
				}
				have += ")"
				want := "("
				for i, p := range m.node.Parameters {
					if i > 0 {
						want += ", "
					}
					want += p.Name
					if i == wantSize-1 && isVariadic {
						want += "..."
					}
				}
				want += ")"
				name := node.Macro.Name
				if node.Import != nil {
					name = node.Import.Name + " " + name
				}
				if haveSize < wantSize {
					err = s.errorf(node, "not enough arguments in show of %s\n\thave %s\n\twant %s", name, have, want)
				} else {
					err = s.errorf(node, "too many arguments in show of %s\n\thave %s\n\twant %s", name, have, want)
				}
				if s.handleError(err) {
					continue
				}
				return err
			}

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
				arg, err := s.eval(argument)
				if err != nil {
					if s.handleError(err) {
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
			st := &rendering{
				scope:       s.scope,
				path:        m.path,
				vars:        []scope{s.vars[0], s.vars[1], s.scope[m.path], arguments},
				treeContext: s.treeContext,
				handleError: s.handleError,
			}
			err = st.render(wr, m.node.Body, nil)
			if err != nil {
				return err
			}

		case *ast.Import:

			path := node.Tree.Path
			if _, ok := s.scope[path]; !ok {
				st := &rendering{
					scope:       s.scope,
					path:        path,
					vars:        []scope{s.vars[0], s.vars[1], {}},
					treeContext: s.treeContext,
					handleError: s.handleError,
				}
				err := st.render(nil, node.Tree.Nodes, nil)
				if err != nil {
					return err
				}
				s.scope[path] = st.vars[2]
				for name, m := range st.vars[2] {
					if _, ok := m.(macro); !ok {
						continue
					}
					if strings.Index(name, ".") > 0 {
						continue
					}
					if fc, _ := utf8.DecodeRuneInString(name); !unicode.Is(unicode.Lu, fc) {
						continue
					}
					s.vars[2][name] = m
				}
			}

		case *ast.ShowPath:

			s.vars = append(s.vars, nil)
			st := &rendering{
				scope:       s.scope,
				path:        node.Tree.Path,
				vars:        s.vars,
				treeContext: s.treeContext,
				handleError: s.handleError,
			}
			err := st.render(wr, node.Tree.Nodes, nil)
			s.vars = s.vars[:len(s.vars)-1]
			if err != nil {
				return err
			}

		}
	}

	return nil
}

// variable returns the value of the variable name in rendering s.
func (s *rendering) variable(name string) (interface{}, bool) {
	for i := len(s.vars) - 1; i >= 0; i-- {
		if s.vars[i] != nil {
			if v, ok := s.vars[i][name]; ok {
				return v, true
			}
		}
	}
	return nil, false
}

func (s *rendering) decimalToInt(node ast.Node, d decimal.Decimal) (int, error) {
	if d.LessThan(minInt) || maxInt.LessThan(d) {
		return 0, s.errorf(node, "number %s overflows int", d)
	}
	p := d.IntPart()
	if !decimal.New(p, 0).Equal(d) {
		return 0, s.errorf(node, "number %s truncated to integer", d)
	}
	return int(p), nil
}

func typeof(v interface{}) string {
	if v == nil {
		return "nil"
	}
	v = asBase(v)
	switch v.(type) {
	case int, decimal.Decimal:
		return "number"
	case string, HTML:
		return "string"
	case bool:
		return "bool"
	default:
		rv := reflect.ValueOf(v)
		rt := rv.Type()
		switch rt.Kind() {
		case reflect.Slice:
			return "slice"
		case reflect.Map, reflect.Ptr:
			return "struct"
		}
	}
	return fmt.Sprintf("(%T)", v)
}
