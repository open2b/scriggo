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
				var ok bool
				c, ok = expr.(bool)
				if !ok {
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

type address interface {
	assign(value interface{}) error
}

type blankAddress struct{}

func (addr blankAddress) assign(interface{}) error {
	return nil
}

type scopeAddress struct {
	Scope scope
	Var   string
}

func (addr scopeAddress) assign(value interface{}) error {
	addr.Scope[addr.Var] = value
	return nil
}

type mapAddress struct {
	Map Map
	Key interface{}
}

func (addr mapAddress) assign(value interface{}) error {
	addr.Map.Store(addr.Key, value)
	return nil
}

type sliceAddress struct {
	Slice Slice
	Index int
}

func (addr sliceAddress) assign(value interface{}) error {
	addr.Slice[addr.Index] = value
	return nil
}

type bytesAddress struct {
	Bytes Bytes
	Index int
	Var   ast.Expression
	Expr  ast.Expression // Expr is nil in multiple assignment.
}

func (addr bytesAddress) assign(value interface{}) error {
	if b, ok := value.(byte); ok {
		addr.Bytes[addr.Index] = b
		return nil
	}
	var b byte
	var err error
	switch n := asBase(value).(type) {
	case int:
		b, err = intToByte(n)
	case *apd.Decimal:
		b, err = decimalToByte(n)
	default:
		if addr.Expr == nil {
			err = fmt.Errorf("cannot assign %s to %s (type byte) in multiple assignment", typeof(n), addr.Var)
		} else {
			err = fmt.Errorf("cannot use %s (type %s) as type byte in assignment", addr.Expr, typeof(n))
		}
	}
	addr.Bytes[addr.Index] = b
	return err
}

func (r *rendering) address(variable, expression ast.Expression) (address, error) {
	var addr address
	switch v := variable.(type) {
	case *ast.Identifier:
		if v.Name == "_" {
			return blankAddress{}, nil
		}
		for j := len(r.vars) - 1; j >= 0; j-- {
			if vars := r.vars[j]; vars != nil {
				if vv, ok := vars[v.Name]; ok {
					if j == 0 {
						if vt, ok := vv.(valuetype); ok {
							return nil, r.errorf(variable, "type %s is not an expression", vt)
						}
						if v != nil && reflect.TypeOf(v).Kind() == reflect.Func {
							return nil, r.errorf(v, "use of builtin %s not in function call", v.Name)
						}
						return nil, r.errorf(v, "cannot assign to %s", v.Name)
					}
					if j == 1 {
						return nil, r.errorf(v, "cannot assign to %s", v.Name)
					}
					if m, ok := vv.(macro); ok {
						return nil, r.errorf(v, "cannot assign to a macro (macro %s declared at %s:%s)",
							v.Name, m.path, m.node.Pos())
					}
					addr = scopeAddress{Scope: vars, Var: v.Name}
					break
				}
			}
		}
		if addr == nil {
			return nil, r.errorf(v, "variable %s not declared", v.Name)
		}
	case *ast.Selector:
		value, err := r.eval(v.Expr)
		if err != nil {
			return nil, err
		}
		m, ok := value.(Map)
		if !ok {
			if typeof(value) == "map" {
				return nil, r.errorf(variable, "cannot assign to a non-mutable map")
			}
			return nil, r.errorf(variable, "cannot assign to %s", variable)
		}
		addr = mapAddress{Map: m, Key: v.Ident}
	case *ast.Index:
		value, err := r.eval(v.Expr)
		if err != nil {
			return nil, err
		}
		switch val := value.(type) {
		case Map:
			key := asBase(r.evalExpression(v.Index))
			switch key.(type) {
			case nil, string, HTML, *apd.Decimal, int, bool:
			default:
				return nil, r.errorf(variable, "hash of unhashable type %s", typeof(key))
			}
			addr = mapAddress{Map: val, Key: key}
		case Slice:
			index, err := r.sliceIndex(v.Index)
			if err != nil {
				return nil, err
			}
			addr = sliceAddress{Slice: val, Index: index}
		case Bytes:
			index, err := r.sliceIndex(v.Index)
			if err != nil {
				return nil, err
			}
			addr = bytesAddress{Bytes: val, Index: index, Var: variable, Expr: expression}
		default:
			switch t := typeof(value); t {
			case "map", "slice", "bytes":
				return nil, r.errorf(v, "cannot assign to a non-mutable %s", t)
			default:
				return nil, r.errorf(v, "invalid operation: %s (type %s does not support indexing)", variable, t)
			}
		}
	}
	return addr, nil
}

func (r *rendering) addresses(node *ast.Assignment) ([]address, error) {

	// addresses contains the addresses of the variables to be assigned.
	addresses := make([]address, len(node.Variables))

	if node.Type == ast.AssignmentDeclaration {

		var vars scope
		if r.vars[len(r.vars)-1] == nil {
			r.vars[len(r.vars)-1] = scope{}
		}
		vars = r.vars[len(r.vars)-1]
		var newVariables bool
		for i, variable := range node.Variables {
			ident := variable.(*ast.Identifier)
			if ident.Name == "_" {
				addresses[i] = blankAddress{}
				continue
			}
			if v, ok := vars[ident.Name]; ok {
				if m, ok := v.(macro); ok {
					return nil, r.errorf(ident, "cannot assign to a macro (macro %s declared at %s:%s)",
						ident.Name, m.path, m.node.Pos())
				}
			} else {
				newVariables = true
			}
			addresses[i] = scopeAddress{Scope: vars, Var: ident.Name}
		}
		if !newVariables {
			return nil, r.errorf(node, "no new variables on left side of :=")
		}

	} else {

		var err error
		if len(node.Variables) == 1 {
			addresses[0], err = r.address(node.Variables[0], node.Expr)
		} else {
			for i, variable := range node.Variables {
				addresses[i], err = r.address(variable, nil)
			}
		}
		if err != nil {
			return nil, err
		}

	}

	return addresses, nil
}

func (r *rendering) renderAssignment(node *ast.Assignment) error {

	switch node.Type {
	case ast.AssignmentIncrement:
		address, err := r.address(node.Variables[0], node.Expr)
		if err != nil {
			return err
		}
		v, err := r.eval(node.Variables[0])
		if err != nil {
			return err
		}
		switch e := v.(type) {
		case byte:
			err = address.assign(e + 1)
		default:
			switch e := asBase(v).(type) {
			case int:
				if ee := e + 1; ee < e {
					d := apd.New(int64(e), 0)
					_, _ = decimalContext.Add(d, d, decimal1)
					err = address.assign(d)
				} else {
					err = address.assign(ee)
				}
			case *apd.Decimal:
				d := new(apd.Decimal)
				_, _ = decimalContext.Add(d, e, decimal1)
				err = address.assign(d)
			case nil:
				err = fmt.Errorf("invalid operation: %s (increment of nil)", node)
			default:
				err = fmt.Errorf("invalid operation: %s (non-numeric type %s)", node, typeof(v))
			}
		}
		if err != nil {
			return r.errorf(node, "%s", err)
		}
	case ast.AssignmentDecrement:
		address, err := r.address(node.Variables[0], node.Expr)
		if err != nil {
			return err
		}
		v, err := r.eval(node.Variables[0])
		if err != nil {
			return err
		}
		switch e := v.(type) {
		case byte:
			err = address.assign(e - 1)
		default:
			switch e := asBase(v).(type) {
			case int:
				if ee := e - 1; ee > e {
					x := apd.New(int64(e), 0)
					_, _ = decimalContext.Sub(x, x, decimal1)
					err = address.assign(x)
				} else {
					err = address.assign(ee)
				}
			case *apd.Decimal:
				x := new(apd.Decimal)
				_, _ = decimalContext.Sub(x, e, decimal1)
				err = address.assign(x)
			case nil:
				return r.errorf(node, "invalid operation: %s (decrement of nil)", node)
			default:
				err = r.errorf(node, "invalid operation: %s (non-numeric type %s)", node, typeof(v))
			}
		}
		if err != nil {
			return r.errorf(node, "%s", err)
		}
	default:
		addresses, err := r.addresses(node)
		if err != nil {
			return err
		}
		switch len(node.Variables) {
		case 1:
			v, err := r.eval(node.Expr)
			if err != nil {
				return err
			}
			err = addresses[0].assign(v)
			if err != nil {
				return r.errorf(node, "%s", err)
			}
		case 2:
			v0, v1, err := r.eval2(node.Expr)
			if err != nil {
				return err
			}
			err = addresses[0].assign(v0)
			if err != nil {
				return r.errorf(node, "%s", err)
			}
			err = addresses[1].assign(v1)
			if err != nil {
				return r.errorf(node, "%s", err)
			}
		default:
			values, err := r.evalN(node.Expr, len(node.Variables))
			if err != nil {
				return err
			}
			for i, v := range values {
				err = addresses[i].assign(v.Interface())
				if err != nil {
					return r.errorf(node, "%s", err)
				}
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

func decimalToInt(n *apd.Decimal) (int, error) {
	if n.Cmp(decimalMinInt) == -1 || n.Cmp(decimalMaxInt) == 1 {
		return 0, fmt.Errorf("number %s overflows int", n)
	}
	p, err := n.Int64()
	if err != nil {
		return 0, fmt.Errorf("number %s truncated to integer", n)
	}
	return int(p), nil
}

func decimalToByte(n *apd.Decimal) (byte, error) {
	if n.Cmp(decimalMinByte) == -1 || decimalMaxByte.Cmp(n) == -1 {
		return 0, fmt.Errorf("number %s overflows byte", n)
	}
	p, err := n.Int64()
	if err != nil {
		return 0, fmt.Errorf("number %s truncated to byte", n)
	}
	return byte(p), nil
}

func intToByte(n int) (byte, error) {
	if n < 0 || n > 255 {
		return 0, fmt.Errorf("number %d overflows byte", n)
	}
	return byte(n), nil
}

func typeof(v interface{}) string {
	if v == nil {
		return "nil"
	}
	v = asBase(v)
	switch v.(type) {
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
