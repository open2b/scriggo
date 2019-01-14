// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"fmt"
	"reflect"

	"github.com/cockroachdb/apd"

	"open2b/template/ast"
)

func (r *rendering) renderAssignment(node *ast.Assignment) error {

	switch node.Type {
	case ast.AssignmentSimple, ast.AssignmentDeclaration:
		addresses, err := r.addresses(node)
		if err != nil {
			return err
		}
		switch len(node.Variables) {
		case 1:
			v, err := r.eval(node.Values[0])
			if err != nil {
				return err
			}
			err = addresses[0].assign(v)
			if err != nil {
				return r.errorf(node, "%s", err)
			}
		case 2:
			var value1 ast.Expression
			if len(node.Values) > 1 {
				value1 = node.Values[1]
			}
			v0, v1, err := r.eval2(node.Values[0], value1)
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
			values, err := r.evalN(node.Values, len(node.Variables))
			if err != nil {
				return err
			}
			for i, v := range values {
				err = addresses[i].assign(v)
				if err != nil {
					return r.errorf(node, "%s", err)
				}
			}
		}
	case ast.AssignmentIncrement, ast.AssignmentDecrement:
		address, err := r.address(node.Variables[0], nil)
		if err != nil {
			return err
		}
		v := address.value()
		switch e := v.(type) {
		case byte:
			switch node.Type {
			case ast.AssignmentIncrement:
				err = address.assign(e + 1)
			case ast.AssignmentDecrement:
				err = address.assign(e - 1)
			}
		case zero:
			if node.Type == ast.AssignmentIncrement {
				_ = address.assign(1)
			} else {
				_ = address.assign(-1)
			}
		default:
			switch e := asBase(v).(type) {
			case nil:
				err = fmt.Errorf("cannot assign to nil")
			case int:
				switch node.Type {
				case ast.AssignmentIncrement:
					if ee := e + 1; ee < e {
						d := apd.New(int64(e), 0)
						_, _ = decimalContext.Add(d, d, decimal1)
						err = address.assign(d)
					} else {
						err = address.assign(ee)
					}
				case ast.AssignmentDecrement:
					if ee := e - 1; ee > e {
						d := apd.New(int64(e), 0)
						_, _ = decimalContext.Sub(d, d, decimal1)
						err = address.assign(d)
					} else {
						err = address.assign(ee)
					}
				}
			case *apd.Decimal:
				switch node.Type {
				case ast.AssignmentIncrement:
					d := new(apd.Decimal)
					_, _ = decimalContext.Add(d, e, decimal1)
					err = address.assign(d)
				case ast.AssignmentDecrement:
					d := new(apd.Decimal)
					_, _ = decimalContext.Sub(d, e, decimal1)
					err = address.assign(d)
				}
			default:
				err = fmt.Errorf("invalid operation: %s (non-numeric type %s)", node, typeof(e))
			}
		}
		if err != nil {
			return r.errorf(node, "%s", err)
		}
	default:
		address, err := r.address(node.Variables[0], nil)
		if err != nil {
			return err
		}
		var v interface{}
		v1 := address.value()
		v2, err := r.eval(node.Values[0])
		if err != nil {
			return err
		}
		v2 = asBase(v2)
		switch node.Type {
		case ast.AssignmentAddition:
			v, err = r.evalAddition(v1, v2)
			if _, ok := v.(zero); ok && err == nil {
				return r.errorf(node, "operands are both untyped zero")
			}
		case ast.AssignmentSubtraction:
			v, err = r.evalSubtraction(v1, v2)
		case ast.AssignmentMultiplication:
			v, err = r.evalMultiplication(v1, v2)
		case ast.AssignmentDivision:
			v, err = r.evalDivision(v1, v2)
		case ast.AssignmentModulo:
			v, err = r.evalModulo(v1, v2)
		}
		if err != nil {
			return r.errorf(node, "invalid operation: %s (%s)", node, err)
		}
		_ = address.assign(v)
	}

	return nil
}

type address interface {
	assign(value interface{}) error
	value() interface{}
}

type blankAddress struct{}

func (addr blankAddress) assign(interface{}) error {
	return nil
}

func (addr blankAddress) value() interface{} {
	return nil
}

type scopeAddress struct {
	Scope scope
	Var   string
}

func (addr scopeAddress) assign(value interface{}) error {
	if _, ok := value.(zero); ok {
		value = nil
	}
	addr.Scope[addr.Var] = value
	return nil
}

func (addr scopeAddress) value() interface{} {
	return addr.Scope[addr.Var]
}

type mapAddress struct {
	Map Map
	Key interface{}
}

func (addr mapAddress) assign(value interface{}) error {
	if _, ok := value.(zero); ok {
		value = nil
	}
	addr.Map.Store(addr.Key, value)
	return nil
}

func (addr mapAddress) value() interface{} {
	if value, ok := addr.Map.Load(addr.Key); ok {
		return value
	}
	return zero{}
}

type sliceAddress struct {
	Slice Slice
	Index int
}

func (addr sliceAddress) assign(value interface{}) error {
	if _, ok := value.(zero); ok {
		value = nil
	}
	addr.Slice[addr.Index] = value
	return nil
}

func (addr sliceAddress) value() interface{} {
	return addr.Slice[addr.Index]
}

type bytesAddress struct {
	Bytes Bytes
	Index int
	Var   ast.Expression
	Expr  ast.Expression // Value is nil in multiple assignment.
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
		b = byte(n)
	case *apd.Decimal:
		b = decimalToByte(n)
	case zero:
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

func (addr bytesAddress) value() interface{} {
	return addr.Bytes[addr.Index]
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
		switch m := value.(type) {
		case Map:
			if m == nil {
				return nil, r.errorf(variable, "assignment to entry in nil map")
			}
			addr = mapAddress{Map: m, Key: v.Ident}
		default:
			if typeof(value) == "map" {
				return nil, r.errorf(variable, "cannot assign to a non-mutable map")
			}
			return nil, r.errorf(variable, "cannot assign to %s", variable)
		}
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
			if val == nil {
				return nil, r.errorf(variable, "assignment to entry in nil map")
			}
			addr = mapAddress{Map: val, Key: key}
		case Slice:
			index, err := r.sliceIndex(v.Index)
			if err != nil {
				return nil, err
			}
			if index >= len(val) {
				return nil, r.errorf(variable, "index out of range")
			}
			addr = sliceAddress{Slice: val, Index: index}
		case Bytes:
			index, err := r.sliceIndex(v.Index)
			if err != nil {
				return nil, err
			}
			if index >= len(val) {
				return nil, r.errorf(variable, "index out of range")
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
			addresses[0], err = r.address(node.Variables[0], node.Values[0])
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
