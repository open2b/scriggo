// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"errors"
	"fmt"
	"reflect"

	"scriggo/internal/compiler/ast"
)

var errTypeConversion = errors.New("failed type conversion")

const constPrecision = 512

const (
	maxInt   = int(maxUint >> 1)
	minInt   = -maxInt - 1
	maxInt64 = 1<<63 - 1
	minInt64 = -1 << 63
	maxUint  = ^uint(0)
)

var boolOperators = [21]bool{
	ast.OperatorEqual:    true,
	ast.OperatorNotEqual: true,
	ast.OperatorAndAnd:   true,
	ast.OperatorOrOr:     true,
}

var intOperators = [21]bool{
	ast.OperatorEqual:          true,
	ast.OperatorNotEqual:       true,
	ast.OperatorLess:           true,
	ast.OperatorLessOrEqual:    true,
	ast.OperatorGreater:        true,
	ast.OperatorGreaterOrEqual: true,
	ast.OperatorAddition:       true,
	ast.OperatorSubtraction:    true,
	ast.OperatorMultiplication: true,
	ast.OperatorDivision:       true,
	ast.OperatorModulo:         true,
	ast.OperatorAnd:            true,
	ast.OperatorOr:             true,
	ast.OperatorXor:            true,
	ast.OperatorAndNot:         true,
	ast.OperatorLeftShift:      true,
	ast.OperatorRightShift:     true,
}

var floatOperators = [21]bool{
	ast.OperatorEqual:          true,
	ast.OperatorNotEqual:       true,
	ast.OperatorLess:           true,
	ast.OperatorLessOrEqual:    true,
	ast.OperatorGreater:        true,
	ast.OperatorGreaterOrEqual: true,
	ast.OperatorAddition:       true,
	ast.OperatorSubtraction:    true,
	ast.OperatorMultiplication: true,
	ast.OperatorDivision:       true,
}

var stringOperators = [21]bool{
	ast.OperatorEqual:          true,
	ast.OperatorNotEqual:       true,
	ast.OperatorLess:           true,
	ast.OperatorLessOrEqual:    true,
	ast.OperatorGreater:        true,
	ast.OperatorGreaterOrEqual: true,
	ast.OperatorAddition:       true,
}

var interfaceOperators = [21]bool{
	ast.OperatorEqual:    true,
	ast.OperatorNotEqual: true,
}

var evalToBoolOperators = [21]bool{
	ast.OperatorEqual:          true,
	ast.OperatorNotEqual:       true,
	ast.OperatorLess:           true,
	ast.OperatorLessOrEqual:    true,
	ast.OperatorGreater:        true,
	ast.OperatorGreaterOrEqual: true,
	ast.OperatorAndAnd:         true,
	ast.OperatorOrOr:           true,
}

var operatorsOfKind = [...][21]bool{
	reflect.Bool:      boolOperators,
	reflect.Int:       intOperators,
	reflect.Int8:      intOperators,
	reflect.Int16:     intOperators,
	reflect.Int32:     intOperators,
	reflect.Int64:     intOperators,
	reflect.Uint:      intOperators,
	reflect.Uint8:     intOperators,
	reflect.Uint16:    intOperators,
	reflect.Uint32:    intOperators,
	reflect.Uint64:    intOperators,
	reflect.Float32:   floatOperators,
	reflect.Float64:   floatOperators,
	reflect.String:    stringOperators,
	reflect.Interface: interfaceOperators,
}

// convert converts a value explicitly. If the converted value is a constant,
// convert returns its value, otherwise returns nil.
//
// If the value can not be converted, returns an errTypeConversion type error,
// errConstantTruncated or errConstantOverflow.
func convert(ti *TypeInfo, t2 reflect.Type) (constant, error) {

	t := ti.Type
	k2 := t2.Kind()

	if ti.Nil() {
		switch t2.Kind() {
		case reflect.Ptr, reflect.Func, reflect.Slice, reflect.Map, reflect.Chan, reflect.Interface:
			return nil, nil
		}
		return nil, errTypeConversion
	}

	if ti.IsConstant() && k2 != reflect.Interface {
		k1 := t.Kind()
		if k2 == reflect.String && isInteger(k1) {
			// As a special case, an integer constant can be explicitly
			// converted to a string type.
			if n := ti.Constant.representedBy(reflect.Int64); n != nil {
				return stringConst(n.int64()), nil
			}
			return stringConst("\uFFFD"), nil
		} else if k2 == reflect.Slice && k1 == reflect.String {
			// As a special case, a string constant can be explicitly converted
			// to a slice of runes or bytes.
			if elem := t2.Elem(); elem == uint8Type || elem == int32Type {
				return nil, nil
			}
		}
		return representedBy(ti, t2)
	}

	if t.ConvertibleTo(t2) {
		return nil, nil
	}

	return nil, errTypeConversion
}

// emptyMethodSet reports whether an interface has an empty method set.
func emptyMethodSet(iface reflect.Type) bool {
	return iface == emptyInterfaceType || boolType.Implements(iface)
}

// fieldByName returns the struct field with the given name and a boolean
// indicating if the field was found.
func fieldByName(t *TypeInfo, name string) (*TypeInfo, bool) {
	if t.Type.Kind() == reflect.Struct {
		field, ok := t.Type.FieldByName(name)
		if ok {
			return &TypeInfo{Type: field.Type}, true
		}
	}
	if t.Type.Kind() == reflect.Ptr {
		field, ok := t.Type.Elem().FieldByName(name)
		if ok {
			return &TypeInfo{Type: field.Type}, true
		}
	}
	return nil, false
}

// fillParametersTypes takes a list of parameters (function arguments or
// function return values) and "fills" their types. For instance, a function
// arguments signature "a, b int" becocmes "a int, b int".
func fillParametersTypes(params []*ast.Field) {
	if len(params) == 0 {
		return
	}
	typ := params[len(params)-1].Type
	for i := len(params) - 1; i >= 0; i-- {
		if params[i].Type != nil {
			typ = params[i].Type
		}
		params[i].Type = typ
	}
	return
}

// isAssignableTo reports whether x is assignable to type t.
// See https://golang.org/ref/spec#Assignability for details.
func isAssignableTo(x *TypeInfo, t reflect.Type) bool {
	if x.Type == t {
		return true
	}
	k := t.Kind()
	if x.Nil() {
		switch k {
		case reflect.Ptr, reflect.Func, reflect.Slice, reflect.Map, reflect.Chan, reflect.Interface:
			return true
		}
		return false
	}
	if k == reflect.Interface {
		return x.Type.Implements(t)
	}
	if x.IsConstant() {
		return x.Constant.representedBy(k) != nil
	}
	// Checks if the type of x and t have identical underlying types and at
	// least one is not a defined type.
	return x.Type.AssignableTo(t)
}

// isBlankIdentifier reports whether expr is an identifier representing the
// blank identifier "_".
func isBlankIdentifier(expr ast.Expression) bool {
	ident, ok := expr.(*ast.Identifier)
	return ok && ident.Name == "_"
}

// isComparison reports whether op is a comparison operator.
func isComparison(op ast.OperatorType) bool {
	return op >= ast.OperatorEqual && op <= ast.OperatorGreaterOrEqual
}

// isComplex reports whether a reflect kind is complex.
func isComplex(k reflect.Kind) bool {
	return k == reflect.Complex64 || k == reflect.Complex128
}

// isInteger reports whether a reflect kind is integer.
func isInteger(k reflect.Kind) bool {
	return reflect.Int <= k && k <= reflect.Uint64
}

// isNumeric reports whether a reflect kind is numeric.
func isNumeric(k reflect.Kind) bool {
	return reflect.Int <= k && k <= reflect.Complex128
}

// isOrdered reports whether t is ordered.
func isOrdered(t *TypeInfo) bool {
	k := t.Type.Kind()
	return isNumeric(k) || k == reflect.String
}

// macroToFunc converts a macro node into a function node.
func macroToFunc(macro *ast.Macro) *ast.Func {
	return ast.NewFunc(macro.Pos(), macro.Ident, macro.Type, ast.NewBlock(nil, macro.Body))
}

type receiverTransformation int

const (
	receiverNoTransform receiverTransformation = iota
	receiverAddAddress
	receiverAddIndirect
)

// methodByName returns a function type that describe the method with that
// name and a boolean indicating if the method was found.
//
// Only for type classes, the returned function type has the method's
// receiver as first argument.
func methodByName(t *TypeInfo, name string) (*TypeInfo, receiverTransformation, bool) {

	if t.IsType() {
		// Method expression on interface type.
		if t.Type.Kind() == reflect.Interface {
			if method, ok := t.Type.MethodByName(name); ok {
				in := make([]reflect.Type, method.Type.NumIn()+1)
				in[0] = t.Type
				for i := 0; i < method.Type.NumIn(); i++ {
					in[i+1] = method.Type.In(i)
				}
				out := make([]reflect.Type, method.Type.NumOut())
				for i := 0; i < method.Type.NumOut(); i++ {
					out[i] = method.Type.Out(i)
				}
				f := func(args []reflect.Value) []reflect.Value {
					return args[0].MethodByName(method.Name).Call(args[1:])
				}
				methExpr := reflect.MakeFunc(reflect.FuncOf(in, out, method.Type.IsVariadic()), f)
				ti := &TypeInfo{
					Type:       removeEnvArg(methExpr.Type(), false),
					Value:      methExpr,
					Properties: PropertyIsPredefined,
				}
				return ti, receiverNoTransform, true
			}
		}

		// Method expression on concrete type.
		if method, ok := t.Type.MethodByName(name); ok {
			return &TypeInfo{
				Type:       removeEnvArg(method.Type, true),
				Value:      method.Func,
				Properties: PropertyIsPredefined,
			}, receiverNoTransform, true
		}
		return nil, receiverNoTransform, false
	}

	// Method calls and method values on interfaces.
	if t.Type.Kind() == reflect.Interface {
		if method, ok := t.Type.MethodByName(name); ok {
			ti := &TypeInfo{
				Type:       removeEnvArg(method.Type, true),
				Value:      name,
				MethodType: MethodValueInterface,
			}
			return ti, receiverNoTransform, true
		}
		return nil, receiverNoTransform, false
	}

	// Method calls and method values on concrete types.
	method := reflect.Zero(t.Type).MethodByName(name)
	methodExplicitRcvr, _ := t.Type.MethodByName(name)
	if method.IsValid() {
		ti := &TypeInfo{
			Type:       removeEnvArg(method.Type(), false),
			Value:      methodExplicitRcvr.Func,
			Properties: PropertyIsPredefined,
			MethodType: MethodValueConcrete,
		}
		// Checks if pointer is defined on T or *T when called on a *T receiver.
		if t.Type.Kind() == reflect.Ptr {
			// TODO(Gianluca): always check first if t has method m, regardless
			// of whether it's a pointer receiver or not. If m exists, it's
			// done. Else, if receiver was a pointer, call MethodByName on t
			// (which is a pointer).
			if method, methodOnT := t.Type.Elem().MethodByName(name); methodOnT {
				// Needs indirection: x.m -> (*x).m
				ti.Type = removeEnvArg(reflect.Zero(t.Type).MethodByName(name).Type(), false)
				ti.Value = method.Func
				return ti, receiverAddIndirect, true
			}
		}
		return ti, receiverNoTransform, true
	}
	if t.Type.Kind() != reflect.Ptr {
		method = reflect.Zero(reflect.PtrTo(t.Type)).MethodByName(name)
		methodExplicitRcvr, _ := reflect.PtrTo(t.Type).MethodByName(name)
		if method.IsValid() {
			return &TypeInfo{
				Type:       removeEnvArg(method.Type(), false),
				Value:      methodExplicitRcvr.Func,
				Properties: PropertyIsPredefined,
				MethodType: MethodValueConcrete,
			}, receiverAddAddress, true
		}
	}
	return nil, receiverNoTransform, false
}

// removeEnvArg returns a type equal to typ but with the vm environment
// parameter removed, if there is one. hasReceiver reports whether the first
// argument of typ is a receiver.
func removeEnvArg(typ reflect.Type, hasReceiver bool) reflect.Type {
	numIn := typ.NumIn()
	if hasReceiver && (numIn <= 1 || typ.In(1) != envType) {
		return typ
	}
	if !hasReceiver && (numIn == 0 || typ.In(0) != envType) {
		return typ
	}
	ins := make([]reflect.Type, numIn-1)
	if hasReceiver {
		ins[0] = typ.In(0)
		for i := 2; i < numIn; i++ {
			ins[i-1] = typ.In(i)
		}
	} else {
		for i := 1; i < numIn; i++ {
			ins[i-1] = typ.In(i)
		}
	}
	outs := make([]reflect.Type, typ.NumOut())
	for i := range outs {
		outs[i] = typ.Out(i)
	}
	return reflect.FuncOf(ins, outs, typ.IsVariadic())
}

// representedBy returns t1 ( a constant or an untyped boolean value )
// represented as a value of type t2. t2 can not be an interface.
func representedBy(t1 *TypeInfo, t2 reflect.Type) (constant, error) {
	k := t2.Kind()
	if t1.IsConstant() {
		c := t1.Constant.representedBy(k)
		if c == nil {
			switch {
			case reflect.Int <= k && k <= reflect.Int64:
				return nil, fmt.Errorf("constant %v truncated to integer", t1.Constant)
			case reflect.Uint <= k && k <= reflect.Uintptr || k == reflect.Float32 || k == reflect.Float64:
				return nil, fmt.Errorf("constant %v overflows %s", t1.Constant, t2)
			}
			return nil, fmt.Errorf("cannot convert %v (type %s) to type %s", t1.Constant, t1, t2)
		}
		return c, nil
	}
	if k != reflect.Bool {
		return nil, fmt.Errorf("cannot use unsigned bool as type %s in assignment", t2)
	}
	return nil, nil
}

// typedValue returns a constant type info value represented with a given
// type.
func typedValue(ti *TypeInfo, t reflect.Type) interface{} {
	k := t.Kind()
	if k == reflect.Interface {
		t = ti.Type
		k = t.Kind()
	}
	c := ti.Constant
	if t.Name() == "" {
		switch k {
		case reflect.Bool:
			return c.bool()
		case reflect.String:
			return c.string()
		case reflect.Int:
			return int(c.int64())
		case reflect.Int8:
			return int8(c.int64())
		case reflect.Int16:
			return int16(c.int64())
		case reflect.Int32:
			return int32(c.int64())
		case reflect.Int64:
			return c.int64()
		case reflect.Uint:
			return uint(c.uint64())
		case reflect.Uint8:
			return uint8(c.uint64())
		case reflect.Uint16:
			return uint16(c.uint64())
		case reflect.Uint32:
			return uint32(c.uint64())
		case reflect.Uint64:
			return c.uint64()
		case reflect.Float32:
			return float32(c.float64())
		case reflect.Float64:
			return c.float64()
		case reflect.Complex64:
			return c.complex128()
		case reflect.Complex128:
			return complex64(c.complex128())
		default:
			panic(fmt.Sprintf("unexpected kind %q", k))
		}
	}
	nv := reflect.New(t).Elem()
	switch k {
	case reflect.Bool:
		nv.SetBool(c.bool())
	case reflect.String:
		nv.SetString(c.string())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		nv.SetInt(c.int64())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		nv.SetUint(c.uint64())
	case reflect.Float32, reflect.Float64:
		nv.SetFloat(c.float64())
	case reflect.Complex64, reflect.Complex128:
		nv.SetComplex(c.complex128())
	default:
		panic("unexpected kind")
	}
	return nv.Interface()
}
