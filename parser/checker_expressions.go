// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"errors"
	"fmt"
	"go/constant"
	gotoken "go/token"
	"math/big"
	"reflect"

	"scrigo/ast"
)

const noEllipses = -1

// TODO (Gianluca): implement with a bits array (int32?), and use bit
// operations to read/write.

var reflectOfDefaultType = [...]reflect.Type{
	reflect.Int:        universe["int"].Type,
	reflect.Int32:      universe["rune"].Type,
	reflect.Float64:    universe["float64"].Type,
	reflect.Complex128: universe["complex128"].Type,
	reflect.String:     universe["string"].Type,
	reflect.Bool:       universe["bool"].Type,
}

var numericKind = [...]bool{
	reflect.Int:           true,
	reflect.Int8:          true,
	reflect.Int16:         true,
	reflect.Int32:         true,
	reflect.Int64:         true,
	reflect.Uint:          true,
	reflect.Uint8:         true,
	reflect.Uint16:        true,
	reflect.Uint32:        true,
	reflect.Uint64:        true,
	reflect.Float32:       true,
	reflect.Float64:       true,
	reflect.UnsafePointer: false,
}

var boolOperators = [15]bool{
	ast.OperatorEqual:    true,
	ast.OperatorNotEqual: true,
	ast.OperatorAnd:      true,
	ast.OperatorOr:       true,
}

var intOperators = [15]bool{
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
}

var floatOperators = [15]bool{
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

var stringOperators = [15]bool{
	ast.OperatorEqual:          true,
	ast.OperatorNotEqual:       true,
	ast.OperatorLess:           true,
	ast.OperatorLessOrEqual:    true,
	ast.OperatorGreater:        true,
	ast.OperatorGreaterOrEqual: true,
	ast.OperatorAddition:       true,
}

var operatorsOfKind = [...][15]bool{
	reflect.Bool:    boolOperators,
	reflect.Int:     intOperators,
	reflect.Int8:    intOperators,
	reflect.Int16:   intOperators,
	reflect.Int32:   intOperators,
	reflect.Int64:   intOperators,
	reflect.Uint:    intOperators,
	reflect.Uint8:   intOperators,
	reflect.Uint16:  intOperators,
	reflect.Uint32:  intOperators,
	reflect.Uint64:  intOperators,
	reflect.Float32: floatOperators,
	reflect.Float64: floatOperators,
	reflect.String:  stringOperators,
}

type typeCheckerScope map[string]*ast.TypeInfo

type HTML string

var boolType = reflect.TypeOf(false)
var intType = reflect.TypeOf(0)

var builtinTypeInfo = &ast.TypeInfo{Properties: ast.PropertyIsBuiltin}
var uint8TypeInfo = &ast.TypeInfo{Type: reflect.TypeOf(int8(0)), Properties: ast.PropertyIsType}
var int32TypeInfo = &ast.TypeInfo{Type: reflect.TypeOf(int32(0)), Properties: ast.PropertyIsType}

var universe = typeCheckerScope{
	"append":     builtinTypeInfo,
	"cap":        builtinTypeInfo,
	"close":      builtinTypeInfo,
	"complex":    builtinTypeInfo,
	"copy":       builtinTypeInfo,
	"delete":     builtinTypeInfo,
	"imag":       builtinTypeInfo,
	"len":        builtinTypeInfo,
	"make":       builtinTypeInfo,
	"new":        builtinTypeInfo,
	"panic":      builtinTypeInfo,
	"print":      builtinTypeInfo,
	"println":    builtinTypeInfo,
	"real":       builtinTypeInfo,
	"recover":    builtinTypeInfo,
	"byte":       uint8TypeInfo,
	"bool":       &ast.TypeInfo{Type: boolType, Properties: ast.PropertyIsType},
	"complex128": &ast.TypeInfo{Type: reflect.TypeOf(complex128(0)), Properties: ast.PropertyIsType},
	"complex64":  &ast.TypeInfo{Type: reflect.TypeOf(complex64(0)), Properties: ast.PropertyIsType},
	"error":      &ast.TypeInfo{Type: reflect.TypeOf((*error)(nil)), Properties: ast.PropertyIsType},
	"float32":    &ast.TypeInfo{Type: reflect.TypeOf(float32(0)), Properties: ast.PropertyIsType},
	"float64":    &ast.TypeInfo{Type: reflect.TypeOf(float64(0)), Properties: ast.PropertyIsType},
	"false":      &ast.TypeInfo{Value: &ast.UntypedValue{DefaultType: reflect.Bool, Bool: false}},
	"int":        &ast.TypeInfo{Type: intType, Properties: ast.PropertyIsType},
	"int16":      &ast.TypeInfo{Type: reflect.TypeOf(int16(0)), Properties: ast.PropertyIsType},
	"int32":      int32TypeInfo,
	"int64":      &ast.TypeInfo{Type: reflect.TypeOf(int64(0)), Properties: ast.PropertyIsType},
	"int8":       uint8TypeInfo,
	"rune":       int32TypeInfo,
	"string":     &ast.TypeInfo{Type: reflect.TypeOf(""), Properties: ast.PropertyIsType},
	"true":       &ast.TypeInfo{Value: &ast.UntypedValue{DefaultType: reflect.Bool, Bool: true}},
	"uint":       &ast.TypeInfo{Type: reflect.TypeOf(uint(0)), Properties: ast.PropertyIsType},
	"uint16":     &ast.TypeInfo{Type: reflect.TypeOf(uint32(0)), Properties: ast.PropertyIsType},
	"uint32":     &ast.TypeInfo{Type: reflect.TypeOf(uint32(0)), Properties: ast.PropertyIsType},
	"uint64":     &ast.TypeInfo{Type: reflect.TypeOf(uint64(0)), Properties: ast.PropertyIsType},
	"uint8":      &ast.TypeInfo{Type: reflect.TypeOf(uint8(0)), Properties: ast.PropertyIsType},
	"uintptr":    &ast.TypeInfo{Type: reflect.TypeOf(uintptr(0)), Properties: ast.PropertyIsType},
}

type funcBound struct {
	bound int
	node  *ast.Func
}

// typechecker represents the state of a type checking.
type typechecker struct {
	path         string
	imports      map[string]*Package // TODO (Gianluca): review!
	fileBlock    typeCheckerScope
	packageBlock typeCheckerScope
	scopes       []typeCheckerScope
	funcBounds   []*funcBound
}

// AddScope adds a new empty scope to the type checker.
func (tc *typechecker) AddScope() {
	tc.scopes = append(tc.scopes, make(typeCheckerScope))
}

// RemoveCurrentScope removes the current scope from the type checker.
func (tc *typechecker) RemoveCurrentScope() {
	tc.scopes = tc.scopes[:len(tc.scopes)-1]
}

// LookupScope looks up name in the scopes. Returns the type info of the name or
// false if the name does not exist. If justCurrentScope is true, LookupScope
// looks up only in the current scope.
func (tc *typechecker) LookupScope(name string, justCurrentScope bool) (*ast.TypeInfo, bool) {
	if justCurrentScope {
		for n, ti := range tc.scopes[len(tc.scopes)-1] {
			if n == name {
				return ti, true
			}
		}
		return nil, false
	}
	for i := len(tc.scopes) - 1; i >= 0; i-- {
		for n, ti := range tc.scopes[i] {
			if n == name {
				return ti, true
			}
		}
	}
	return nil, false
}

// AssignScope assigns value to name in the last scope.
func (tc *typechecker) AssignScope(name string, value *ast.TypeInfo) {
	tc.scopes[len(tc.scopes)-1][name] = value
}

func (tc *typechecker) CheckUpValue(name string) string {
	bound := tc.funcBounds[len(tc.funcBounds)-1].bound
	for i := len(tc.scopes) - 1; i >= 0; i-- {
		for n, _ := range tc.scopes[i] {
			if n != name {
				continue
			}
			if i < bound-1 { // out of current function scope.
				tc.scopes[i][n].Properties |= ast.PropertyMustBeReferenced
				return name
			}
			return ""
		}
	}
	return ""
}

// TODO (Gianluca): check if using all declared identifiers.
func (tc *typechecker) checkIdentifier(ident *ast.Identifier) *ast.TypeInfo {
	if len(tc.funcBounds) > 0 {
		uv := tc.CheckUpValue(ident.Name)
		if uv != "" {
			lastFunc := tc.funcBounds[len(tc.funcBounds)-1]
			lastFunc.node.Upvalues = append(lastFunc.node.Upvalues, uv)
		}
	}
	i, ok := tc.LookupScope(ident.Name, false)
	if !ok {
		panic(tc.errorf(ident, "undefined: %s", ident.Name))
	}
	return i
}

// errorf builds and returns a type check error.
func (tc *typechecker) errorf(nodeOrPos interface{}, format string, args ...interface{}) error {
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
		Path: tc.path,
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

// checkExpression returns the type info of expr. Returns an error if expr is
// a type or a package.
func (tc *typechecker) checkExpression(expr ast.Expression) *ast.TypeInfo {
	ti := tc.typeof(expr, noEllipses)
	if ti.IsType() || ti.IsPackage() {
		panic(tc.errorf(expr, "%s is not an expression", ti))
	}
	expr.SetTypeInfo(ti)
	return ti
}

// checkType evaluates expr as a type and returns the type info. Returns an
// error if expr is not an type.
func (tc *typechecker) checkType(expr ast.Expression, length int) *ast.TypeInfo {
	ti := tc.typeof(expr, length)
	if !ti.IsType() {
		panic(tc.errorf(expr, "%s is not a type", ti))
	}
	expr.SetTypeInfo(ti)
	return ti
}

// typeof returns the type of expr. If expr is not an expression but a type,
// returns the type.
func (tc *typechecker) typeof(expr ast.Expression, length int) *ast.TypeInfo {

	switch expr := expr.(type) {

	case *ast.String:
		return &ast.TypeInfo{
			Value: &ast.UntypedValue{
				DefaultType: reflect.String,
				String:      expr.Text,
			},
		}

	case *ast.Int:
		return &ast.TypeInfo{
			Value: &ast.UntypedValue{
				DefaultType: reflect.Int,
				Number:      constant.MakeFromLiteral(expr.Value.String(), gotoken.INT, 0),
			},
		}

	case *ast.Rune:
		return &ast.TypeInfo{
			Value: &ast.UntypedValue{
				DefaultType: reflect.Int32,
				Number:      constant.MakeInt64(int64(expr.Value)),
			},
		}

	case *ast.Float:
		return &ast.TypeInfo{
			Value: &ast.UntypedValue{
				DefaultType: reflect.Float64,
				Number:      constant.MakeFromLiteral(expr.Value.Text('f', -1), gotoken.FLOAT, 0),
			},
		}

	case *ast.Parentesis:
		panic("unexpected parentesis")

	case *ast.UnaryOperator:
		t := tc.checkExpression(expr.Expr)
		if t.Nil() {
			panic(tc.errorf(expr, "invalid operation: ! nil"))
		}
		switch expr.Op {
		case ast.OperatorNot:
			if c := t.Value; c != nil {
				if c.(*ast.UntypedValue).DefaultType != reflect.Bool {
					panic(tc.errorf(expr, "invalid operation: ! %s", expr))
				}
				c = &ast.UntypedValue{DefaultType: c.(*ast.UntypedValue).DefaultType, Bool: !c.(*ast.UntypedValue).Bool}
				t = &ast.TypeInfo{Properties: t.Properties, Type: t.Type, Value: c}
			} else if t.Type != nil && t.Type.Kind() != reflect.Bool {
				panic(tc.errorf(expr, "invalid operation: ! %s", expr))
			}
			return t
		case ast.OperatorAddition, ast.OperatorSubtraction:
			if c := t.Value; c != nil {
				dt := c.(*ast.UntypedValue).DefaultType
				if dt == reflect.String || dt == reflect.Bool {
					panic(tc.errorf(expr, "invalid operation: %s %s", expr.Op, t))
				}
				if expr.Op == ast.OperatorSubtraction {
					c = &ast.UntypedValue{DefaultType: dt, Number: constant.UnaryOp(gotoken.SUB, c.(*ast.UntypedValue).Number, 0)}
					t = &ast.TypeInfo{Properties: t.Properties, Type: t.Type, Value: c}
				}
			} else if t.Type != nil && !numericKind[t.Type.Kind()] {
				panic(tc.errorf(expr, "invalid operation: %s %s", expr.Op, t))
			}
			return t
		}

	case *ast.BinaryOperator:
		t1 := tc.checkExpression(expr.Expr1)
		t2 := tc.checkExpression(expr.Expr2)
		if t1.Nil() && t2.Nil() {
			panic(tc.errorf(expr, "invalid operation: %v (operator %s not defined on nil)", expr, expr.Op))
		}
		if t1.Nil() || t2.Nil() {
			var t = t1
			if t.Nil() {
				t = t2
			}
			if t.Type != nil && !t.Type.Comparable() {
				panic(tc.errorf(expr, "cannot convert nil to type %s", t))
			}
			if expr.Op != ast.OperatorEqual && expr.Op != ast.OperatorNotEqual {
				panic(tc.errorf(expr, "invalid operation: %v (operator %s not defined on %s)", expr, expr.Op, t.Type.Kind()))
			}
			return &ast.TypeInfo{}
		}
		if t1.Value != nil && t2.Value != nil {
			return tc.binaryOp(expr)
		}
		if t2.Type == nil {
			_, err := tc.convert(t2, t1.Type, false)
			if err != nil {
				if err == errTypeConversion {
					panic(tc.errorf(expr, "cannot convert %v (type %s) to type %s", expr, t2, t1))
				}
				panic(tc.errorf(expr, "%s", err))
			}
			t2 = &ast.TypeInfo{Type: t1.Type}
		} else if t1.Type == nil {
			_, err := tc.convert(t2, t2.Type, false)
			if err != nil {
				if err == errTypeConversion {
					panic(tc.errorf(expr, "cannot convert %v (type %s) to type %s", expr, t1, t2))
				}
				panic(tc.errorf(expr, "%s", err))
			}
			t1 = &ast.TypeInfo{Type: t2.Type}
		}
		if expr.Op >= ast.OperatorEqual && expr.Op <= ast.OperatorGreaterOrEqual {
			if !tc.isAssignableTo(t1, t2.Type) && !tc.isAssignableTo(t2, t1.Type) {
				panic(tc.errorf(expr, "invalid operation: %v (mismatched types %s and %s)", expr, t1.ShortString(), t2.ShortString()))
			}
			if expr.Op == ast.OperatorEqual || expr.Op == ast.OperatorNotEqual {
				if !tc.isComparable(t1) {
					// TODO(marco) explain in the error message why they are not comparable.
					panic(tc.errorf(expr, "invalid operation: %v (%s cannot be compared)", expr, t1.Type))
				}
			} else if !tc.isOrdered(t1) {
				panic(tc.errorf(expr, "invalid operation: %v (operator %s not defined on %s)", expr, expr.Op, t1.Type.Kind()))
			}
			return &ast.TypeInfo{}
		}
		if t1.Type != t2.Type {
			panic(tc.errorf(expr, "invalid operation: %v (mismatched types %s and %s)", expr, t1.ShortString(), t2.ShortString()))
		}
		if kind := t1.Type.Kind(); !operatorsOfKind[kind][expr.Op] {
			panic(tc.errorf(expr, "invalid operation: %v (operator %s not defined on %s)", expr, expr.Op, kind))
		}
		return t1

	case *ast.Identifier:
		t := tc.checkIdentifier(expr)
		if t.IsPackage() {
			panic(tc.errorf(expr, "use of package %s without selector", t))
		}
		return t

	case *ast.MapType:
		key := tc.checkType(expr.KeyType, noEllipses)
		value := tc.checkType(expr.ValueType, noEllipses)
		defer func() {
			if rec := recover(); rec != nil {
				panic(tc.errorf(expr, "invalid map key type %s", key))
			}
		}()
		return &ast.TypeInfo{Properties: ast.PropertyIsType, Type: reflect.MapOf(key.Type, value.Type)}

	case *ast.SliceType:
		elem := tc.checkType(expr.ElementType, noEllipses)
		return &ast.TypeInfo{Properties: ast.PropertyIsType, Type: reflect.SliceOf(elem.Type)}

	case *ast.ArrayType:
		elem := tc.checkType(expr.ElementType, noEllipses)
		if expr.Len == nil { // ellipsis.
			return &ast.TypeInfo{Properties: ast.PropertyIsType, Type: reflect.ArrayOf(length, elem.Type)}
		}
		len := tc.checkExpression(expr.Len)
		if len.Value == nil { // TODO (Gianluca): should check if it's a constant
			panic(tc.errorf(expr, "non-constant array bound %s", expr.Len))
		}
		declLength, err := tc.convert(len, intType, false)
		if err != nil {
			panic(tc.errorf(expr, err.Error()))
		}
		if declLength.(int) < 0 {
			panic(tc.errorf(expr, "array bound must be non-negative"))
		}
		if length > declLength.(int) {
			panic(tc.errorf(expr, "array index %d out of bounds [0:%d]", length-1, declLength.(int)))
		}
		return &ast.TypeInfo{Properties: ast.PropertyIsType, Type: reflect.ArrayOf(length, elem.Type)}

	case *ast.CompositeLiteral:
		elem, err := tc.checkCompositeLiteral(expr, nil)
		// TODO (Gianluca): checkCompositeLiteral should panic, not return errors.
		if err != nil {
			panic(err)
		}
		return &ast.TypeInfo{Type: reflect.TypeOf(elem)}

	case *ast.FuncType:
		variadic := expr.IsVariadic
		// Parameters.
		numIn := len(expr.Parameters)
		in := make([]reflect.Type, numIn)
		for i := numIn - 1; i >= 0; i-- {
			param := expr.Parameters[i]
			if param.Type == nil {
				in[i] = in[i+1]
			} else {
				t := tc.checkType(param.Type, noEllipses)
				if variadic && i == numIn-1 {
					in[i] = reflect.SliceOf(t.Type)
				} else {
					in[i] = t.Type
				}
			}
		}
		// Result.
		numOut := len(expr.Result)
		out := make([]reflect.Type, numOut)
		for i := numOut - 1; i >= 0; i-- {
			res := expr.Result[i]
			if res == nil {
				out[i] = out[i+1]
			} else {
				c := tc.checkType(res.Type, noEllipses)
				out[i] = c.Type
			}
		}
		return &ast.TypeInfo{Type: reflect.FuncOf(in, out, variadic), Properties: ast.PropertyIsType}

	case *ast.Func:
		t := tc.checkType(expr.Type, noEllipses)
		tc.AddScope()
		tc.funcBounds = append(tc.funcBounds, &funcBound{len(tc.scopes), expr})
		tc.checkNodes(expr.Body.Nodes)
		tc.funcBounds = tc.funcBounds[:len(tc.funcBounds)-1]
		tc.RemoveCurrentScope()
		return &ast.TypeInfo{Type: t.Type}

	case *ast.Call:
		types := tc.checkCallExpression(expr, false)
		if len(types) == 0 {
			panic(tc.errorf(expr, "%v used as value", expr))
		}
		if len(types) > 1 {
			panic(tc.errorf(expr, "multiple-value %v in single-value context", expr))
		}

	case *ast.Index:
		t := tc.checkExpression(expr.Expr)
		if t.Nil() {
			panic(tc.errorf(expr, "use of untyped nil"))
		}
		switch kind := t.Kind(); kind {
		case reflect.Slice, reflect.String, reflect.Array, reflect.Ptr:
			k := kind
			if kind == reflect.Ptr {
				if t.Type.Elem().Kind() != reflect.Array {
					panic(tc.errorf(expr, "invalid operation: %v (type %s does not support indexing)", expr, t))
				}
				k = reflect.Array
			}
			index := tc.checkExpression(expr.Index)
			v, err := tc.convert(index, intType, false)
			if err != nil {
				if err == errTypeConversion {
					err = fmt.Errorf("non-integer %s index %s", k, index)
				}
				panic(tc.errorf(expr, "%s", err))
			}
			if v != nil {
				if v.(int) < 0 {
					panic(tc.errorf(expr, "invalid %s index %s (index must be non-negative)", k, index))
				}
				if t.Value != nil {
					if s := constantString(t); len(s) <= v.(int) {
						panic(tc.errorf(expr, "invalid string index %d (out of bounds for %d-byte string)", v, len(s)))
					}
				}
			}
			var typ reflect.Type
			switch kind {
			case reflect.String:
				typ = universe["byte"].Type
			case reflect.Slice, reflect.Array:
				typ = t.Type.Elem()
			case reflect.Ptr:
				typ = t.Type.Elem().Elem()
			}
			if t.Addressable() && kind != reflect.String {
				return &ast.TypeInfo{Type: typ, Properties: ast.PropertyAddressable}
			}
			return &ast.TypeInfo{Type: typ}
		case reflect.Map:
			key := tc.checkExpression(expr.Index)
			if !tc.isAssignableTo(key, t.Type.Key()) {
				if key.Nil() {
					panic(tc.errorf(expr, "cannot convert nil to type %s", t.Type.Key()))
				}
				panic(tc.errorf(expr, "cannot use %v (type %s) as type %s in map index", expr.Index, key, t.Type.Key()))
			}
			return &ast.TypeInfo{Type: t.Type.Elem()}
		default:
			panic(tc.errorf(expr, "invalid operation: %v (type %s does not support indexing)", expr, t.ShortString()))
		}

	case *ast.Slicing:
		// TODO(marco) support full slice expressions
		// TODO(marco) check if an array is addressable
		t := tc.checkExpression(expr.Expr)
		if t.Nil() {
			panic(tc.errorf(expr, "use of untyped nil"))
		}
		kind := t.Type.Kind()
		switch kind {
		case reflect.String, reflect.Slice, reflect.Array:
		default:
			if kind != reflect.Ptr || t.Type.Elem().Kind() != reflect.Array {
				panic(tc.errorf(expr, "cannot slice %v (type %s)", expr.Expr, t))
			}
		}
		if expr.Low != nil {
			low := tc.checkExpression(expr.Low)
			if low.Nil() {
				panic(tc.errorf(expr, "invalid slice index nil (type nil)"))
			}
			if low.Type.Kind() != reflect.Int {
				panic(tc.errorf(expr, "invalid slice index %v (type %s)", expr.Low, low))
			}
		}
		if expr.High != nil {
			high := tc.checkExpression(expr.High)
			if high.Nil() {
				panic(tc.errorf(expr, "invalid slice index nil (type nil)"))
			}
			if high.Type.Kind() != reflect.Int {
				panic(tc.errorf(expr, "invalid slice index %v (type %s)", expr.High, high))
			}
		}
		switch kind {
		case reflect.String, reflect.Slice:
			return t
		case reflect.Array:
			return &ast.TypeInfo{Type: reflect.SliceOf(t.Type.Elem())}
		case reflect.Ptr:
			return &ast.TypeInfo{Type: reflect.SliceOf(t.Type.Elem().Elem())}
		}

	case *ast.Selector:
		t := tc.typeof(expr.Expr, noEllipses)
		if t.IsPackage() {
			// TODO
		}
		if t.IsType() {
			method, ok := tc.methodByName(t, expr.Ident)
			if !ok {
				panic(tc.errorf(expr, "%v undefined (type %s has no method %s)", expr, t, expr.Ident))
			}
			return method
		}
		method, ok := tc.methodByName(t, expr.Ident)
		if ok {
			return method
		}
		field, ok := tc.fieldByName(t, expr.Ident)
		if ok {
			return field
		}
		panic(tc.errorf(expr, "%v undefined (type %s has no field or method %s)", expr, t, expr.Ident))

	case *ast.TypeAssertion:
		t := tc.typeof(expr.Expr, noEllipses)
		if t.Type.Kind() != reflect.Interface {
			panic(tc.errorf(expr, "invalid type assertion: %v (non-interface type %s on left)", expr, t))
		}
		expr.Expr.SetTypeInfo(t)
		if expr.Type == nil {
			return nil
		}
		t = tc.checkType(expr.Type, noEllipses)
		expr.Type.SetTypeInfo(t)
		return t

	}

	panic(fmt.Errorf("unexpected: %v (type %T)", expr, expr))
}

// checkCallExpression type checks a call expression, including type
// conversions and built-in function calls.
func (tc *typechecker) checkCallExpression(expr *ast.Call, statement bool) []*ast.TypeInfo {

	t := tc.typeof(expr.Func, noEllipses)

	if t.Nil() {
		panic(tc.errorf(expr, "use of untyped nil"))
	}

	if t.IsType() {
		if len(expr.Args) == 0 {
			panic(tc.errorf(expr, "missing argument to conversion to %s: %s", t, expr))
		}
		if len(expr.Args) > 1 {
			panic(tc.errorf(expr, "too many arguments to conversion to %s: %s", t, expr))
		}
		arg := tc.checkExpression(expr.Args[0])
		value, err := tc.convert(arg, t.Type, true)
		if err != nil {
			if err == errTypeConversion {
				panic(tc.errorf(expr, "cannot convert %s (type %s) to type %s", expr.Args[0], arg.Type, t.Type))
			}
			panic(tc.errorf(expr, "%s", err))
		}
		return []*ast.TypeInfo{{Type: t.Type, Value: value}}
	}

	if t.IsPackage() {
		panic(tc.errorf(expr, "use of package fmt without selector"))
	}

	if t == builtinTypeInfo {

		ident := expr.Func.(*ast.Identifier)

		switch ident.Name {

		case "append":
			if len(expr.Args) == 0 {
				panic(tc.errorf(expr, "missing arguments to append"))
			}
			slice := tc.checkExpression(expr.Args[0])
			if slice.Nil() {
				panic(tc.errorf(expr, "first argument to append must be typed slice; have untyped nil"))
			}
			if slice.Type.Kind() != reflect.Slice {
				panic(tc.errorf(expr, "first argument to append must be slice; have %s", t))
			}
			if len(expr.Args) > 1 {
				// TODO(marco): implements variadic call to append
				// TODO(marco): implements append([]byte{}, "abc"...)
				elem := t.Type.Elem()
				for _, el := range expr.Args[1:] {
					t := tc.checkExpression(el)
					if !tc.isAssignableTo(t, elem) {
						if t == nil {
							panic(tc.errorf(expr, "cannot use nil as type %s in append", elem))
						}
						panic(tc.errorf(expr, "cannot use %v (type %s) as type %s in append", el, t, elem))
					}
				}
			}
			return []*ast.TypeInfo{slice}

		case "copy":
			if len(expr.Args) < 2 {
				panic(tc.errorf(expr, "missing argument to copy: %s", expr))
			}
			if len(expr.Args) > 2 {
				panic(tc.errorf(expr, "too many arguments to copy: %s", expr))
			}
			dst := tc.checkExpression(expr.Args[0])
			src := tc.checkExpression(expr.Args[1])
			if dst.Nil() || src.Nil() {
				panic(tc.errorf(expr, "use of untyped nil"))
			}
			dk := dst.Type.Kind()
			sk := dst.Type.Kind()
			switch {
			case dk != reflect.Slice && sk != reflect.Slice:
				panic(tc.errorf(expr, "arguments to copy must be slices; have %s, %s", dst, src))
			case dk != reflect.Slice:
				panic(tc.errorf(expr, "first argument to copy should be slice; have %s", dst))
			case sk != reflect.Slice && sk != reflect.String:
				panic(tc.errorf(expr, "second argument to copy should be slice or string; have %s", src))
			case sk == reflect.String && dst.Type.Elem().Kind() != reflect.Uint8:
				panic(tc.errorf(expr, "arguments to copy have different element types: %s and %s", dst, src))
			}
			// TODO(marco): verificare se il confronto dei reflect.typ è sufficiente per essere conformi alle specifiche
			if dk == reflect.Slice && sk == reflect.Slice && dst != src {
				panic(tc.errorf(expr, "arguments to copy have different element types: %s and %s", dst, src))
			}
			return []*ast.TypeInfo{{Type: intType}}

		case "delete":
			if len(expr.Args) < 2 {
				panic(tc.errorf(expr, "missing argument to delete: %s", expr))
			}
			if len(expr.Args) > 2 {
				panic(tc.errorf(expr, "too many arguments to delete: %s", expr))
			}
			t := tc.checkExpression(expr.Args[0])
			key := tc.checkExpression(expr.Args[1])
			if t.Nil() {
				panic(tc.errorf(expr, "first argument to delete must be map; have nil"))
			}
			if t.Type.Kind() != reflect.Map {
				panic(tc.errorf(expr, "first argument to delete must be map; have %s", t))
			}
			if rkey := t.Type.Key(); !tc.isAssignableTo(key, rkey) {
				if key == nil {
					panic(tc.errorf(expr, "cannot use nil as type %s in delete", rkey))
				}
				panic(tc.errorf(expr, "cannot use %v (type %s) as type %s in delete", expr.Args[1], key, rkey))
			}
			return nil

		case "len":
			if len(expr.Args) < 1 {
				panic(tc.errorf(expr, "missing argument to len: %s", expr))
			}
			if len(expr.Args) > 1 {
				panic(tc.errorf(expr, "too many arguments to len: %s", expr))
			}
			t := tc.checkExpression(expr.Args[0])
			if t.Nil() {
				panic(tc.errorf(expr, "use of untyped nil"))
			}
			switch k := t.Type.Kind(); k {
			case reflect.String, reflect.Array, reflect.Map, reflect.Chan:
			default:
				if k != reflect.Ptr || t.Type.Elem().Kind() != reflect.Array {
					panic(tc.errorf(expr, "invalid argument %v (type %s) for len", expr, t))
				}
			}
			return []*ast.TypeInfo{{Type: intType}}

		case "make":
			numArgs := len(expr.Args)
			if numArgs == 0 {
				panic(tc.errorf(expr, "missing argument to make"))
			}
			t := tc.typeof(expr.Args[0], noEllipses)
			if !t.IsType() {
				panic(tc.errorf(expr.Args[0], "%s is not a type", expr.Args[0]))
			}
			var ok bool
			switch t.Type.Kind() {
			case reflect.Slice:
				if numArgs == 1 {
					panic(tc.errorf(expr, "missing len argument to make(%s)", expr.Args[0]))
				}
				if numArgs > 1 {
					var lenvalue int
					n := tc.checkExpression(expr.Args[1])
					if !n.Nil() {
						if n.Type == nil {
							c, err := tc.convert(n, intType, false)
							if err != nil {
								panic(tc.errorf(expr, "%s", err))
							}
							if ok && c != nil {
								if lenvalue = c.(int); lenvalue < 0 {
									panic(tc.errorf(expr, "negative len argument in make(%s)", expr.Args[0]))
								}
							}
						} else if n.Type.Kind() == reflect.Int {
							ok = true
						}
					}
					if !ok {
						panic(tc.errorf(expr, "non-integer len argument in make(%s) - %s", expr.Args[0], n))
					}
					if numArgs > 2 {
						var capvalue int
						m := tc.checkExpression(expr.Args[2])
						if !m.Nil() {
							if m.Type == nil {
								v, err := tc.convert(m, intType, false)
								if err != nil {
									panic(err)
								}
								if capvalue = v.(int); capvalue < 0 {
									panic(tc.errorf(expr, "negative cap argument in make(%s)", expr.Args[0]))
								}
								if lenvalue > capvalue {
									panic(tc.errorf(expr, "len larger than cap in in make(%s)", expr.Args[0]))
								}
							} else if m.Type.Kind() == reflect.Int {
								ok = true
							}
						}
						if !ok {
							panic(tc.errorf(expr, "non-integer cap argument in make(%s) - %s", expr.Args[0], m))
						}
					}
				}
				if numArgs > 3 {
					panic(tc.errorf(expr, "too many arguments to make(%s)", expr.Args[0]))
				}
			case reflect.Map:
				if numArgs > 2 {
					panic(tc.errorf(expr, "too many arguments to make(%s)", expr.Args[0]))
				}
				if numArgs == 2 {
					n := tc.checkExpression(expr.Args[1])
					if !n.Nil() {
						if n.Type == nil {
							v, err := tc.convert(n, intType, false)
							if err != nil {
								panic(err)
							}
							if ok && v.(int) < 0 {
								panic(tc.errorf(expr, "negative len argument in make(%s)", expr.Args[0]))
							}
						} else if n.Type.Kind() == reflect.Int {
							ok = true
						}
					}
					if !ok {
						panic(tc.errorf(expr, "non-integer size argument in make(%s) - %s", expr.Args[0], n))
					}
				}
			default:
				panic(tc.errorf(expr, "cannot make type %s", t))
			}
			return []*ast.TypeInfo{t}

		case "new":
			if len(expr.Args) == 0 {
				panic(tc.errorf(expr, "missing argument to new"))
			}
			t := tc.typeof(expr.Args[0], noEllipses)
			if t.IsType() {
				panic(tc.errorf(expr, "%s is not a type", expr.Args[0]))
			}
			if len(expr.Args) > 1 {
				panic(tc.errorf(expr, "too many arguments to new(%s)", expr.Args[0]))
			}
			return []*ast.TypeInfo{{Type: reflect.PtrTo(t.Type)}}

		case "panic":
			if len(expr.Args) == 0 {
				panic(tc.errorf(expr, "missing argument to panic: panic()"))
			}
			if len(expr.Args) > 1 {
				panic(tc.errorf(expr, "too many arguments to panic: %s", expr))
			}
			_ = tc.checkExpression(expr.Args[0])
			return nil

		}

		panic(fmt.Sprintf("unexpected builtin %s", ident.Name))

	}

	if t.Type.Kind() != reflect.Func {
		panic(tc.errorf(expr, "cannot call non-function %v (type %s)", expr.Func, t))
	}

	var numIn = t.Type.NumIn()
	var variadic = t.Type.IsVariadic()

	if (!variadic && len(expr.Args) != numIn) || (variadic && len(expr.Args) < numIn-1) {
		have := "("
		for i, arg := range expr.Args {
			if i > 0 {
				have += ", "
			}
			c := tc.checkExpression(arg)
			if c == nil {
				have += "nil"
			} else {
				have += t.String()
			}
		}
		have += ")"
		want := "("
		for i := 0; i < numIn; i++ {
			if i > 0 {
				want += ", "
			}
			if i == numIn-1 && variadic {
				want += "..."
			}
			want += t.Type.In(i).String()
		}
		want += ")"
		if len(expr.Args) < numIn {
			panic(tc.errorf(expr, "not enough arguments in call to %s\n\thave %s\n\twant %s", expr.Func, have, want))
		}
		panic(tc.errorf(expr, "too many arguments in call to %s\n\thave %s\n\twant %s", expr.Func, have, want))
	}

	var lastIn = numIn - 1
	var in reflect.Type

	for i, arg := range expr.Args {
		if i < lastIn || !variadic {
			in = t.Type.In(i)
		} else if i == lastIn {
			in = t.Type.In(lastIn).Elem()
		}
		a := tc.checkExpression(arg)
		if !tc.isAssignableTo(a, in) {
			panic(tc.errorf(expr.Args[i], "cannot use %s (type %s) as type %s in argument to %s", expr.Args[i], a, in, expr.Func))
		}
	}

	numOut := t.Type.NumOut()
	resultTypes := make([]*ast.TypeInfo, numOut)
	for i := 0; i < numOut; i++ {
		resultTypes[i] = &ast.TypeInfo{Type: t.Type.Out(i)}
	}

	return resultTypes
}

// binaryOp executes the binary expression c op u, where c and u are constants.
// The tow operands must be both numeric, boolean or string.
// Panics if it can not be executed.
func (tc *typechecker) binaryOp(expr *ast.BinaryOperator) *ast.TypeInfo {
	c1 := expr.Expr1.TypeInfo().Value.(*ast.UntypedValue)
	c2 := expr.Expr2.TypeInfo().Value.(*ast.UntypedValue)
	mismatch := false
	dt1 := c1.DefaultType
	dt2 := c2.DefaultType
	switch dt1 {
	default:
		mismatch = dt2 == reflect.String || dt2 == reflect.Bool
	case reflect.String:
		mismatch = dt2 != reflect.String
	case reflect.Bool:
		mismatch = dt2 != reflect.Bool
	}
	if mismatch {
		panic(tc.errorf(expr, "invalid operation: %v (mismatched types %s and %s)", expr, dt1, dt2))
	}
	c := ast.UntypedValue{}
	switch dt1 {
	default:
		var v interface{}
		switch expr.Op {
		case ast.OperatorEqual:
			v = constant.Compare(c1.Number, gotoken.EQL, c2.Number)
		case ast.OperatorNotEqual:
			v = constant.Compare(c1.Number, gotoken.NEQ, c2.Number)
		case ast.OperatorLess:
			v = constant.Compare(c1.Number, gotoken.LSS, c2.Number)
		case ast.OperatorLessOrEqual:
			v = constant.Compare(c1.Number, gotoken.LEQ, c2.Number)
		case ast.OperatorGreater:
			v = constant.Compare(c1.Number, gotoken.GTR, c2.Number)
		case ast.OperatorGreaterOrEqual:
			v = constant.Compare(c1.Number, gotoken.GEQ, c2.Number)
		case ast.OperatorAddition:
			v = constant.BinaryOp(c1.Number, gotoken.ADD, c2.Number)
		case ast.OperatorSubtraction:
			v = constant.BinaryOp(c1.Number, gotoken.SUB, c2.Number)
		case ast.OperatorMultiplication:
			v = constant.BinaryOp(c1.Number, gotoken.MUL, c2.Number)
		case ast.OperatorDivision:
			if constant.Sign(c2.Number) == 0 {
				panic(errDivisionByZero)
			}
			if dt1 == reflect.Float64 || dt2 == reflect.Float64 {
				v = constant.BinaryOp(c1.Number, gotoken.QUO, c2.Number)
			} else {
				a, _ := new(big.Int).SetString(c1.Number.ExactString(), 10)
				b, _ := new(big.Int).SetString(c2.Number.ExactString(), 10)
				v = constant.MakeFromLiteral(a.Quo(a, b).String(), gotoken.INT, 0)
			}
		case ast.OperatorModulo:
			if dt1 == reflect.Float64 || dt2 == reflect.Float64 {
				panic(errFloatModulo)
			}
			if constant.Sign(c2.Number) == 0 {
				panic(errDivisionByZero)
			}
			v = constant.BinaryOp(c1.Number, gotoken.REM, c2.Number)
		}
		if number, ok := v.(constant.Value); ok {
			c.DefaultType = dt1
			if c.DefaultType < dt2 {
				c.DefaultType = dt2
			}
			c.Number = number
		} else {
			c.DefaultType = reflect.Bool
			c.Bool = v.(bool)
		}
	case reflect.String:
		switch expr.Op {
		case ast.OperatorEqual, ast.OperatorNotEqual:
			c.DefaultType = reflect.Bool
			c.Bool = c1.String == c2.String
			if expr.Op == ast.OperatorNotEqual {
				c.Bool = !c.Bool
			}
		case ast.OperatorAddition:
			c.DefaultType = reflect.String
			c.String = c1.String + c2.String
		}
	case reflect.Bool:
		c.DefaultType = reflect.Bool
		switch expr.Op {
		case ast.OperatorEqual:
			c.Bool = c1.Bool == c2.Bool
		case ast.OperatorNotEqual:
			c.Bool = c1.Bool != c2.Bool
		case ast.OperatorAnd:
			c.Bool = c1.Bool && c2.Bool
		case ast.OperatorOr:
			c.Bool = c1.Bool || c2.Bool
		default:
			panic(tc.errorf(expr, "invalid operation: %v (operator %s not defined on %s)", expr, expr.Op, dt1))
		}
	}
	return &ast.TypeInfo{Value: &c}
}

var errTypeConversion = errors.New("failed type conversion")

// convert converts a value. explicit reports whether the conversion is
// explicit. If the value is a constant and the type to convert is a concrete
// type, the converted value is a typed constant and its value is returned,
// otherwise returns nil.
//
// If the value can not be converted, returns an error errTypeConversion,
// errConstantTruncated or errConstantOverflow.
func (tc *typechecker) convert(ti *ast.TypeInfo, t2 reflect.Type, explicit bool) (interface{}, error) {

	t := ti.Type
	v := ti.Value
	k2 := t2.Kind()

	if ti.Nil() {
		switch t2.Kind() {
		case reflect.Ptr, reflect.Func, reflect.Slice, reflect.Map, reflect.Chan, reflect.Interface:
			return nil, nil
		}
		return nil, errTypeConversion
	}

	if t == nil {
		if v == nil {
			t = boolType
		} else {
			t = reflectOfDefaultType[v.(*ast.UntypedValue).DefaultType]
		}
	}

	// First, Reflect must return true.
	if !t.ConvertibleTo(t2) {
		return nil, errTypeConversion
	}

	// The special case is only for explicit conversion.
	if !explicit {
		if k := t.Kind(); k2 == reflect.String && numericKind[k] {
			return nil, errTypeConversion
		}
	}

	// For interface types the returned value is not a constant.
	if k2 == reflect.Interface {
		return nil, nil
	}

	// If it is not a constant, returns.
	if v == nil {
		return nil, nil
	}

	// If the resulted value is a constant, returns its value.
	switch c := v.(type) {
	case *ast.UntypedValue:
		switch dt := c.DefaultType; dt {
		case reflect.Bool:
			v = c.Bool
		case reflect.String:
			v = c.String
		default:
			cn := ConstantNumber{val: c.Number, typ: ConstantNumberType(c.DefaultType)}
			var err error
			v, err = cn.ToType(t2)
			if err != nil {
				if _, ok := err.(errConstantConversion); ok {
					err = errTypeConversion
				}
				return nil, err
			}
		}
	}

	return v, nil
}

// fieldByName returns the struct field with the given name and a boolean
// indicating if the field was found.
func (tc *typechecker) fieldByName(t *ast.TypeInfo, name string) (*ast.TypeInfo, bool) {
	field, ok := t.Type.FieldByName(name)
	if ok {
		return &ast.TypeInfo{Type: field.Type}, true
	}
	if t.Type.Kind() == reflect.Ptr {
		field, ok = t.Type.Elem().FieldByName(name)
		if ok {
			return &ast.TypeInfo{Type: field.Type}, true
		}
	}
	return nil, false
}

// TODO (Gianluca): to review.
func (tc *typechecker) defaultType(c *ast.UntypedValue) reflect.Type {
	switch c.DefaultType {
	case reflect.Int:
		return intType
	case reflect.Int32:
		return reflect.TypeOf(rune(0))
	case reflect.Float64:
		return reflect.TypeOf(float64(0))
	case reflect.Complex128:
		return reflect.TypeOf(complex128(0))
	case reflect.String:
		return reflect.TypeOf("")
	case reflect.Bool:
		return reflect.TypeOf(false)
	}
	panic(fmt.Errorf("unexpected default type: %#v", c.DefaultType))
}

// isComparable reports whether t is comparable.
func (tc *typechecker) isComparable(t *ast.TypeInfo) bool {
	if t.Type == nil {
		return true
	}
	return t.Type.Comparable()
}

// isOrdered reports whether t is ordered.
func (tc *typechecker) isOrdered(t *ast.TypeInfo) bool {
	if t.Type == nil {
		// Untyped bool values (constant or not constant) are not ordered.
		switch v := t.Value.(type) {
		case nil, bool:
			return false
		case *ast.UntypedValue:
			return v.DefaultType != reflect.Bool
		}
		return true
	}
	k := t.Type.Kind()
	return numericKind[k] || k == reflect.String
}

// methodByName returns a function type that describe the method with that
// name and a boolean indicating if the method was found.
//
// Only for type classes, the returned function type has the method's
// receiver as first argument.
func (tc *typechecker) methodByName(t *ast.TypeInfo, name string) (*ast.TypeInfo, bool) {
	if t.IsType() {
		if method, ok := t.Type.MethodByName(name); ok {
			return &ast.TypeInfo{Type: method.Type}, true
		}
		return nil, false
	}
	method := reflect.Zero(t.Type).MethodByName(name)
	if method.IsValid() {
		return &ast.TypeInfo{Type: method.Type()}, true
	}
	if t.Type.Kind() != reflect.Ptr {
		method = reflect.Zero(reflect.PtrTo(t.Type)).MethodByName(name)
		if method.IsValid() {
			return &ast.TypeInfo{Type: method.Type()}, true
		}
	}
	return nil, false
}

func constantString(c *ast.TypeInfo) string {
	switch s := c.Value.(type) {
	case string:
		return s
	case *ast.UntypedValue:
		return s.String
	}
	panic("no-string constant")
}
