// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scrigo

import (
	"fmt"
	"math/big"
	"reflect"
	"unicode"
	"unicode/utf8"

	"scrigo/ast"
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
			v, err := r.eval1(node.Values[0])
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
	case ast.AssignmentIncrement:
		address, err := r.address(node.Variables[0], nil)
		if err != nil {
			return err
		}
		v := address.value()
		switch n := v.(type) {
		case int:
			_ = address.assign(n + 1)
		case int64:
			_ = address.assign(n + 1)
		case int32:
			_ = address.assign(n + 1)
		case int16:
			_ = address.assign(n + 1)
		case int8:
			_ = address.assign(n + 1)
		case uint:
			_ = address.assign(n + 1)
		case uint64:
			_ = address.assign(n + 1)
		case uint32:
			_ = address.assign(n + 1)
		case uint16:
			_ = address.assign(n + 1)
		case uint8:
			_ = address.assign(n + 1)
		case float64:
			_ = address.assign(n + 1)
		case float32:
			_ = address.assign(n + 1)
		case ConstantNumber:
			cn, err := n.BinaryOp(ast.OperatorAddition, newConstantInt(big.NewInt(1)))
			if err != nil {
				return err
			}
			_ = address.assign(cn)
		case CustomNumber:
			n.Inc()
		default:
			return r.errorf(node, "invalid operation: %s (non-numeric type %s)", node, typeof(n))
		}
	case ast.AssignmentDecrement:
		address, err := r.address(node.Variables[0], nil)
		if err != nil {
			return err
		}
		v := address.value()
		switch n := v.(type) {
		case int:
			_ = address.assign(n - 1)
		case int64:
			_ = address.assign(n - 1)
		case int32:
			_ = address.assign(n - 1)
		case int16:
			_ = address.assign(n - 1)
		case int8:
			_ = address.assign(n - 1)
		case uint:
			_ = address.assign(n - 1)
		case uint64:
			_ = address.assign(n - 1)
		case uint32:
			_ = address.assign(n - 1)
		case uint16:
			_ = address.assign(n - 1)
		case uint8:
			_ = address.assign(n - 1)
		case float64:
			_ = address.assign(n - 1)
		case float32:
			_ = address.assign(n - 1)
		case ConstantNumber:
			cn, err := n.BinaryOp(ast.OperatorSubtraction, newConstantInt(big.NewInt(1)))
			if err != nil {
				return err
			}
			_ = address.assign(cn)
		case CustomNumber:
			n.Dec()
		default:
			return r.errorf(node, "invalid operation: %s (non-numeric type %s)", node, typeof(n))
		}
	default:
		address, err := r.address(node.Variables[0], nil)
		if err != nil {
			return err
		}
		var v interface{}
		v1 := address.value()
		switch node.Type {
		case ast.AssignmentAddition:
			v, err = r.evalBinary(v1, ast.OperatorAddition, node.Variables[0], node.Values[0])
		case ast.AssignmentSubtraction:
			v, err = r.evalBinary(v1, ast.OperatorSubtraction, node.Variables[0], node.Values[0])
		case ast.AssignmentMultiplication:
			v, err = r.evalBinary(v1, ast.OperatorMultiplication, node.Variables[0], node.Values[0])
		case ast.AssignmentDivision:
			v, err = r.evalBinary(v1, ast.OperatorDivision, node.Variables[0], node.Values[0])
		case ast.AssignmentModulo:
			v, err = r.evalBinary(v1, ast.OperatorModulo, node.Variables[0], node.Values[0])
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
	case int:
		addr.Value.SetInt(int64(v))
	case int64:
		addr.Value.SetInt(v)
	case int32:
		addr.Value.SetInt(int64(v))
	case int16:
		addr.Value.SetInt(int64(v))
	case int8:
		addr.Value.SetInt(int64(v))
	case uint:
		addr.Value.SetUint(uint64(v))
	case uint64:
		addr.Value.SetUint(v)
	case uint32:
		addr.Value.SetUint(uint64(v))
	case uint16:
		addr.Value.SetUint(uint64(v))
	case uint8:
		addr.Value.SetUint(uint64(v))
	case float64:
		addr.Value.SetFloat(v)
	case float32:
		addr.Value.SetFloat(float64(v))
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

type goMapAddress struct {
	Map reflect.Value
	Key reflect.Value
}

func (addr goMapAddress) assign(value interface{}) (err error) {
	addr.Map.SetMapIndex(addr.Key, reflect.ValueOf(value))
	return nil
}

func (addr goMapAddress) value() interface{} {
	if value := addr.Map.MapIndex(addr.Key); value.IsValid() {
		return value.Interface()
	}
	return reflect.Zero(addr.Map.Type().Elem())
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
	if addr.Expr == nil {
		return fmt.Errorf("cannot assign %s to %s (type byte) in multiple assignment", typeof(value), addr.Var)
	}
	return fmt.Errorf("cannot use %s (type %s) as type byte in assignment", addr.Expr, typeof(value))
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
	varsFor:
		for j := len(r.vars) - 1; j >= 0; j-- {
			if vars := r.vars[j]; vars != nil {
				if vv, ok := vars[v.Name]; ok {
					switch vvv := vv.(type) {
					case reference:
						return varAddress{vvv.rv}, nil
					default:
						if j == 0 {
							if vt, ok := vv.(reflect.Type); ok {
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
						break varsFor
					}
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
		case *Package:
			vvv, ok := vv.Declarations[v.Ident]
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
	case *ast.UnaryOperator:
		if v.Operator() != ast.OperatorMultiplication {
			panic(fmt.Sprintf("expected a multiplication operator (*), got operator %v", v.Operator()))
		}
		ptrRaw, err := r.eval(v.Expr)
		if err != nil {
			return nil, err
		}
		if ptrRaw == nil {
			return nil, r.errorf(variable, "nil pointer dereference")
		}
		ptr := reflect.ValueOf(ptrRaw)
		if ptr.Kind() != reflect.Ptr {
			return nil, r.errorf(variable, "invalid indirect of %s (type %s)", v.Expr, ptr.Type())
		}
		addr = varAddress{Value: reflect.Indirect(ptr)}
	case *ast.Index:
		value, err := r.eval(v.Expr)
		if err != nil {
			return nil, err
		}
		switch val := value.(type) {
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
				key, err := r.mapIndex(v.Index, rv.Type().Key())
				if err != nil {
					return nil, err
				}
				addr = goMapAddress{Map: rv, Key: reflect.ValueOf(key)}
			case reflect.Slice:
				index, err := r.sliceIndex(v.Index)
				if err != nil {
					return nil, err
				}
				if index >= rv.Len() {
					return nil, r.errorf(variable, "index out of range")
				}
				addr = goSliceAddress{Slice: rv, Index: index}
			case reflect.Ptr:
				if rv.Elem().Kind() != reflect.Array {
					return nil, r.errorf(v, "invalid operation: %s (type %s does not support indexing)", variable, typeof(variable))
				}
				index, err := r.sliceIndex(v.Index)
				if err != nil {
					return nil, err
				}
				if index >= rv.Elem().Len() {
					return nil, r.errorf(variable, "index out of range")
				}
				addr = goSliceAddress{Slice: rv.Elem(), Index: index}
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
