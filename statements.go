// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
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

	"github.com/cockroachdb/apd"

	"open2b/template/ast"
)

var decimalMaxInt = apd.New(maxInt, 0)
var decimalMinInt = apd.New(minInt, 0)
var decimalMaxRune = apd.New(2147483647, 0)
var decimalMinRune = apd.New(-2147483648, 0)
var decimalMaxByte = apd.New(255, 0)
var decimalMinByte = apd.New(0, 0)
var decimal1 = apd.New(1, 0)
var decimalMod8 = apd.New(256, 0)
var decimalMod32 = apd.New(4294967296, 0)

var decimalModInt *apd.Decimal

func init() {
	if 18446744073709551615 == uint64(^uint(0)) {
		decimalModInt, _, _ = apd.NewFromString("18446744073709551616")
	} else {
		decimalModInt, _, _ = apd.NewFromString("4294967296")
	}
}

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

// render renders nodes.
func (r *rendering) render(wr io.Writer, nodes []ast.Node, urlstate *urlState) error {

Nodes:
	for _, n := range nodes {

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

		case *ast.Value:

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
				switch v := expr.(type) {
				case zero:
				case bool:
					c = v
				default:
					err = r.errorf(node, "non-bool %s (type %s) used as if condition", node.Condition, typeof(expr))
					if !r.handleError(err) {
						return err
					}
				}
			}
			if c {
				if len(node.Then) > 0 {
					err = r.render(wr, node.Then, urlstate)
					if err != nil {
						return err
					}
				}
			} else if len(node.Else) > 0 {
				err = r.render(wr, node.Else, urlstate)
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

		case *ast.Break:
			return errBreak

		case *ast.Continue:
			return errContinue

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
					err = r.errorf(node, "not enough arguments in show of %s\n\thave %s\n\twant %s", name, have, want)
				} else {
					err = r.errorf(node, "too many arguments in show of %s\n\thave %s\n\twant %s", name, have, want)
				}
				if r.handleError(err) {
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

			path := node.Tree.Path
			if _, ok := r.scope[path]; !ok {
				rn := &rendering{
					scope:       r.scope,
					path:        path,
					vars:        []scope{r.vars[0], r.vars[1], {}},
					treeContext: r.treeContext,
					handleError: r.handleError,
				}
				err := rn.render(nil, node.Tree.Nodes, nil)
				if err != nil {
					return err
				}
				r.scope[path] = rn.vars[2]
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

			err := r.eval0(node)
			if err != nil && !r.handleError(err) {
				return err
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
				return v, true
			}
		}
	}
	return nil, false
}

func decimalToInt(n *apd.Decimal) int {
	var d *apd.Decimal
	if n.Exponent == 0 {
		if n.Cmp(decimalMinInt) < 0 || n.Cmp(decimalMaxInt) > 0 {
			d = &apd.Decimal{}
			_, _ = numberConversionContext.Rem(d, n, decimalModInt)
		} else {
			d = n
		}
	} else {
		d = &apd.Decimal{}
		_, _ = numberConversionContext.RoundToIntegralValue(d, n)
		if d.Cmp(decimalMinInt) < 0 || d.Cmp(decimalMaxInt) > 0 {
			_, _ = numberConversionContext.Rem(d, d, decimalModInt)
		}
	}
	p, _ := d.Int64()
	return int(p)
}

func decimalToByte(n *apd.Decimal) byte {
	var d *apd.Decimal
	if n.Exponent == 0 {
		if n.Cmp(decimalMinByte) < 0 || n.Cmp(decimalMaxByte) > 0 {
			d = &apd.Decimal{}
			_, _ = numberConversionContext.Rem(d, n, decimalMod8)
		} else {
			d = n
		}
	} else {
		d = &apd.Decimal{}
		_, _ = numberConversionContext.RoundToIntegralValue(d, n)
		if d.Cmp(decimalMinByte) < 0 || d.Cmp(decimalMaxByte) > 0 {
			_, _ = numberConversionContext.Rem(d, d, decimalMod8)
		}
	}
	p, _ := d.Int64()
	return byte(p)
}

func typeof(v interface{}) string {
	v = asBase(v)
	switch v.(type) {
	case nil:
		return "nil"
	case string, HTML:
		return "string"
	case *apd.Decimal, int, byte:
		return "number"
	case bool:
		return "bool"
	case Map:
		return "map"
	case Slice:
		return "slice"
	case Bytes:
		return "bytes"
	case error:
		return "error"
	case zero:
		return "untyped zero"
	default:
		rt := reflect.TypeOf(v)
		switch rt.Kind() {
		case reflect.Map, reflect.Struct, reflect.Ptr:
			return "map"
		case reflect.Slice:
			if rt.Elem().Kind() == reflect.Uint8 {
				return "bytes"
			}
			return "slice"
		}
	}
	return fmt.Sprintf("(%T)", v)
}
