// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"fmt"
	"reflect"
	"unicode"
	"unicode/utf8"

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
	addr.Scope[addr.Var] = value
	return nil
}

func (addr scopeAddress) value() interface{} {
	return addr.Scope[addr.Var]
}

type varAddress struct {
	Value reflect.Value
}

func (addr varAddress) assign(value interface{}) error {
	switch v := value.(type) {
	case string:
		addr.Value.SetString(v)
	case HTML:
		addr.Value.SetString(string(v))
	case float32:
		addr.Value.SetFloat(float64(v))
	case float64:
		addr.Value.SetFloat(v)
	case int:
		addr.Value.SetInt(int64(v))
	case int8:
		addr.Value.SetInt(int64(v))
	case int16:
		addr.Value.SetInt(int64(v))
	case int32:
		addr.Value.SetInt(int64(v))
	case int64:
		addr.Value.SetInt(v)
	case uint:
		addr.Value.SetUint(uint64(v))
	case uint8:
		addr.Value.SetUint(uint64(v))
	case uint16:
		addr.Value.SetUint(uint64(v))
	case uint32:
		addr.Value.SetUint(uint64(v))
	case uint64:
		addr.Value.SetUint(v)
	case bool:
		addr.Value.SetBool(v)
	case []byte:
		addr.Value.SetBytes(v)
	default:
		addr.Value.Set(reflect.ValueOf(v))
	}
	return nil
}

func (addr varAddress) value() interface{} {
	return reflect.Indirect(addr.Value).Interface()
}

type mapAddress struct {
	Map Map
	Key interface{}
}

func (addr mapAddress) assign(value interface{}) (err error) {
	// Catches unhashable keys errors.
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	addr.Map.Store(addr.Key, value)
	return nil
}

func (addr mapAddress) value() interface{} {
	if value, ok := addr.Map.Load(addr.Key); ok {
		return value
	}
	return nil
}

type goMapAddress struct {
	Map reflect.Value
	Key reflect.Value
}

func (addr goMapAddress) assign(value interface{}) (err error) {
	// Catches unhashable keys errors.
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	addr.Map.SetMapIndex(addr.Key, reflect.ValueOf(value))
	return nil
}

func (addr goMapAddress) value() interface{} {
	if value := addr.Map.MapIndex(addr.Key); value.IsValid() {
		return value.Interface()
	}
	return reflect.Zero(addr.Map.Type().Elem())
}

type sliceAddress struct {
	Slice Slice
	Index int
}

func (addr sliceAddress) assign(value interface{}) error {
	addr.Slice[addr.Index] = value
	return nil
}

func (addr sliceAddress) value() interface{} {
	return addr.Slice[addr.Index]
}

type goSliceAddress struct {
	Slice reflect.Value
	Index int
}

func (addr goSliceAddress) assign(value interface{}) error {
	addr.Slice.Index(addr.Index).Set(reflect.ValueOf(value))
	return nil
}

func (addr goSliceAddress) value() interface{} {
	return addr.Slice.Index(addr.Index).Interface()
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
		switch vv := value.(type) {
		case Map:
			addr = mapAddress{Map: vv, Key: v.Ident}
		case Package:
			vvv, ok := vv[v.Ident]
			if !ok {
				if fc, _ := utf8.DecodeRuneInString(v.Ident); !unicode.Is(unicode.Lu, fc) {
					return nil, r.errorf(variable, "cannot refer to unexported name %s", variable)
				}
				return nil, r.errorf(variable, "undefined: %s", variable)
			}
			rv := reflect.ValueOf(vvv)
			if rv.Kind() != reflect.Ptr {
				return nil, r.errorf(variable, "cannot assign to %s", variable)
			}
			addr = varAddress{Value: rv.Elem()}
		default:
			return nil, r.errorf(variable, "%s undefined (type %s has no field or method %s)", variable, typeof(variable), v.Ident)
		}
	case *ast.Index:
		value, err := r.eval(v.Expr)
		if err != nil {
			return nil, err
		}
		switch val := value.(type) {
		case Map:
			key := r.mapIndex(v.Index)
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
			if index >= len(val) {
				return nil, r.errorf(variable, "index out of range")
			}
			addr = sliceAddress{Slice: val, Index: index}
		case Bytes:
			index, err := r.sliceIndex(v.Index)
			if err != nil {
				return nil, err
			}
			if val == nil {
				return nil, r.errorf(variable, "cannot assign to a non-mutable bytes")
			}
			if index >= len(val) {
				return nil, r.errorf(variable, "index out of range")
			}
			addr = bytesAddress{Bytes: val, Index: index, Var: variable, Expr: expression}
		default:
			rv := reflect.ValueOf(value)
			switch rv.Kind() {
			case reflect.Map:
				key := reflect.ValueOf(r.mapIndex(v.Index))
				addr = goMapAddress{Map: rv, Key: key}
			case reflect.Slice:
				index, err := r.sliceIndex(v.Index)
				if err != nil {
					return nil, err
				}
				if index >= rv.Len() {
					return nil, r.errorf(variable, "index out of range")
				}
				addr = goSliceAddress{Slice: rv, Index: index}
			default:
				return nil, r.errorf(v, "invalid operation: %s (type %s does not support indexing)", variable, typeof(variable))
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
