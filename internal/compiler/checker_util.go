// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/open2b/scriggo/ast"
	"github.com/open2b/scriggo/internal/compiler/types"
	"github.com/open2b/scriggo/native"
)

var envType = reflect.TypeOf((*native.Env)(nil)).Elem()
var errTypeConversion = errors.New("failed type conversion")

type nilConversionError struct {
	typ reflect.Type
}

func (err nilConversionError) Error() string {
	return "cannot convert nil to type " + err.typ.String()
}

const (
	maxInt   = int(maxUint >> 1)
	minInt   = -maxInt - 1
	maxInt64 = 1<<63 - 1
	maxUint  = ^uint(0)
)

// isSigned reports whether kind is a signed integer kind.
func isSigned(kind reflect.Kind) bool {
	return reflect.Int <= kind && kind <= reflect.Int64
}

var boolOperators = [22]bool{
	ast.OperatorEqual:    true,
	ast.OperatorNotEqual: true,
	ast.OperatorAnd:      true,
	ast.OperatorOr:       true,
}

var intOperators = [22]bool{
	ast.OperatorEqual:          true,
	ast.OperatorNotEqual:       true,
	ast.OperatorLess:           true,
	ast.OperatorLessEqual:      true,
	ast.OperatorGreater:        true,
	ast.OperatorGreaterEqual:   true,
	ast.OperatorAddition:       true,
	ast.OperatorSubtraction:    true,
	ast.OperatorMultiplication: true,
	ast.OperatorDivision:       true,
	ast.OperatorModulo:         true,
	ast.OperatorBitAnd:         true,
	ast.OperatorBitOr:          true,
	ast.OperatorXor:            true,
	ast.OperatorAndNot:         true,
	ast.OperatorLeftShift:      true,
	ast.OperatorRightShift:     true,
}

var floatOperators = [22]bool{
	ast.OperatorEqual:          true,
	ast.OperatorNotEqual:       true,
	ast.OperatorLess:           true,
	ast.OperatorLessEqual:      true,
	ast.OperatorGreater:        true,
	ast.OperatorGreaterEqual:   true,
	ast.OperatorAddition:       true,
	ast.OperatorSubtraction:    true,
	ast.OperatorMultiplication: true,
	ast.OperatorDivision:       true,
}

var complexOperators = [22]bool{
	ast.OperatorEqual:          true,
	ast.OperatorNotEqual:       true,
	ast.OperatorAddition:       true,
	ast.OperatorSubtraction:    true,
	ast.OperatorMultiplication: true,
	ast.OperatorDivision:       true,
}

var stringOperators = [22]bool{
	ast.OperatorEqual:        true,
	ast.OperatorNotEqual:     true,
	ast.OperatorLess:         true,
	ast.OperatorLessEqual:    true,
	ast.OperatorGreater:      true,
	ast.OperatorGreaterEqual: true,
	ast.OperatorAddition:     true,
	ast.OperatorContains:     true,
	ast.OperatorNotContains:  true,
}

var interfaceOperators = [22]bool{
	ast.OperatorEqual:    true,
	ast.OperatorNotEqual: true,
}

var evalToBoolOperators = [22]bool{
	ast.OperatorEqual:        true,
	ast.OperatorNotEqual:     true,
	ast.OperatorLess:         true,
	ast.OperatorLessEqual:    true,
	ast.OperatorGreater:      true,
	ast.OperatorGreaterEqual: true,
	ast.OperatorAnd:          true,
	ast.OperatorOr:           true,
	ast.OperatorContains:     true,
	ast.OperatorNotContains:  true,
}

var operatorsOfKind = [...][22]bool{
	reflect.Bool:       boolOperators,
	reflect.Int:        intOperators,
	reflect.Int8:       intOperators,
	reflect.Int16:      intOperators,
	reflect.Int32:      intOperators,
	reflect.Int64:      intOperators,
	reflect.Uint:       intOperators,
	reflect.Uint8:      intOperators,
	reflect.Uint16:     intOperators,
	reflect.Uint32:     intOperators,
	reflect.Uint64:     intOperators,
	reflect.Uintptr:    intOperators,
	reflect.Float32:    floatOperators,
	reflect.Float64:    floatOperators,
	reflect.Complex64:  complexOperators,
	reflect.Complex128: complexOperators,
	reflect.String:     stringOperators,
	reflect.Interface:  interfaceOperators,
}

var constantKindName = map[reflect.Kind]string{
	reflect.Bool:       "boolean",
	reflect.Int:        "integer",
	reflect.Int32:      "rune",
	reflect.Float64:    "floating-point",
	reflect.Complex128: "complex",
}

// convert implicitly converts an untyped value. If the converted value is a
// constant, convert returns its value, otherwise returns nil.
//
// As per spec, untyped values are the predeclared identifier nil, the untyped
// constants and the untyped boolean values.
//
// In addition, convert converts expressions that contain untyped numeric
// constants and shift operations with an untyped constant left operand and a
// non-constant right operand. In this case t2 is the type the left operand
// would assume if the shift expressions were replaced by its left operand
// alone.
func (tc *typechecker) convert(ti *typeInfo, expr ast.Expression, t2 reflect.Type) (constant, error) {

	switch k2 := t2.Kind(); {

	case ti.Nil():
		// predeclared nil.
		switch k2 {
		case reflect.Ptr, reflect.Func, reflect.Slice, reflect.Map, reflect.Chan, reflect.Interface:
			return nil, nil
		}
		return nil, nilConversionError{t2}

	case ti.IsConstant():
		// untyped constant.
		if k2 == reflect.Interface {
			if t2.NumMethod() > 0 {
				return nil, errTypeConversion
			}
			_, err := ti.Constant.representedBy(ti.Type)
			return nil, err
		}
		return ti.Constant.representedBy(t2)

	case ti.Type == boolType:
		// untyped boolean value.
		if k2 == reflect.Bool || (k2 == reflect.Interface && t2.NumMethod() == 0) {
			return nil, nil
		}
		return nil, errTypeConversion

	default:
		// expression that contains untyped numeric constants and shift
		// operations with an untyped constant left operand and a non-constant
		// right operand.

		if !ti.IsNumeric() {
			panic("BUG")
		}

		typ := t2
		if k2 == reflect.Interface {
			if t2.NumMethod() > 0 {
				return nil, errTypeConversion
			}
			typ = ti.Type
		}
		tc.compilation.typeInfos[expr] = &typeInfo{Type: typ}

		switch expr := expr.(type) {

		case *ast.UnaryOperator:
			return tc.convert(tc.compilation.typeInfos[expr.Expr], expr.Expr, typ)

		case *ast.BinaryOperator:
			if op := expr.Operator(); op == ast.OperatorLeftShift || op == ast.OperatorRightShift {
				_, err := tc.convert(tc.compilation.typeInfos[expr.Expr1], expr.Expr1, typ)
				if err != nil {
					return nil, err
				}
				if k := typ.Kind(); k < reflect.Int || k > reflect.Uintptr {
					return nil, fmt.Errorf("invalid operation: %s (shift of type %s)", expr, typ)
				}
				return nil, nil
			}
			_, err := tc.convert(tc.compilation.typeInfos[expr.Expr1], expr.Expr1, typ)
			if err != nil {
				return nil, err
			}
			return tc.convert(tc.compilation.typeInfos[expr.Expr2], expr.Expr2, typ)

		default:
			panic(fmt.Errorf("BUG: unexpected expr %s (type %T) with type info %s", expr, expr, ti))

		}

	}

}

// deferGoBuiltin returns a type info suitable to be embedded into the 'defer'
// and 'go' statements with a builtin call as argument.
func deferGoBuiltin(name string) *typeInfo {
	var fun interface{}
	switch name {
	case "close":
		fun = func(ch interface{}) {
			reflect.ValueOf(ch).Close()
		}
	case "copy":
		fun = func(dst, src interface{}) {
			reflect.Copy(reflect.ValueOf(dst), reflect.ValueOf(src))
		}
	case "delete":
		fun = func(m interface{}, key interface{}) {
			reflect.ValueOf(m).SetMapIndex(reflect.ValueOf(key), reflect.Value{})
		}
	case "panic":
		fun = func(env native.Env, v interface{}) {
			panic(v)
		}
	case "print":
		fun = func(env native.Env, args ...interface{}) {
			env.Print(args...)
		}
	case "println":
		fun = func(env native.Env, args ...interface{}) {
			env.Println(args...)
		}
	case "recover":
		// This native function should only be used with the 'go' statement and
		// not the 'defer' statement.
		fun = func() {}
	}
	rv := reflect.ValueOf(fun)
	return &typeInfo{
		Properties: propertyHasValue | propertyIsNative,
		Type:       removeEnvArg(rv.Type(), false),
		value:      rv,
	}
}

// checkDuplicateParams checks if a function type contains duplicate
// parameter names.
func (tc *typechecker) checkDuplicateParams(fn *ast.FuncType) {
	names := map[string]struct{}{}
	for _, params := range [2][]*ast.Parameter{fn.Parameters, fn.Result} {
		for _, param := range params {
			if param.Ident == nil {
				continue
			}
			if isBlankIdentifier(param.Ident) {
				continue
			}
			name := param.Ident.Name
			if _, ok := names[name]; ok {
				panic(tc.errorf(param.Ident, "duplicate argument %s", name))
			}
			names[name] = struct{}{}
		}
	}
}

type invalidTypeInAssignment string

func (err invalidTypeInAssignment) Error() string {
	return string(err)
}

func newInvalidTypeInAssignment(x *typeInfo, expr ast.Expression, t reflect.Type) invalidTypeInAssignment {
	return invalidTypeInAssignment(fmt.Sprintf("cannot use %s (type %s) as type %s", expr, x, t))
}

// isAssignableTo reports whether x is assignable to type t.
// See https://golang.org/ref/spec#Assignability for details.
func (tc *typechecker) isAssignableTo(x *typeInfo, expr ast.Expression, t reflect.Type) error {
	if x.Untyped() {
		_, err := tc.convert(x, expr, t)
		if err == errNotRepresentable || err == errTypeConversion {
			return newInvalidTypeInAssignment(x, expr, t)
		}
		return err
	}
	if !types.AssignableTo(x.Type, t) {
		return newInvalidTypeInAssignment(x, expr, t)
	}
	return nil
}

// isBlankIdentifier reports whether expr is an identifier representing the
// blank identifier "_".
func isBlankIdentifier(expr ast.Expression) bool {
	ident, ok := expr.(*ast.Identifier)
	return ok && ident.Name == "_"
}

// isPeriodImport reports whether the import node has a period as import name.
func isPeriodImport(impor *ast.Import) bool {
	return impor.Ident != nil && impor.Ident.Name == "."
}

// isBlankImport reports whether the import node has the blank identifier as
// import name.
func isBlankImport(impor *ast.Import) bool {
	return impor.Ident != nil && impor.Ident.Name == "_"
}

// isComparison reports whether op is a comparison operator.
func isComparison(op ast.OperatorType) bool {
	return op >= ast.OperatorEqual && op <= ast.OperatorGreaterEqual ||
		op == ast.OperatorContains || op == ast.OperatorNotContains
}

// isComplex reports whether a reflect kind is complex.
func isComplex(k reflect.Kind) bool {
	return k == reflect.Complex64 || k == reflect.Complex128
}

// isInteger reports whether a reflect kind is integer.
func isInteger(k reflect.Kind) bool {
	return reflect.Int <= k && k <= reflect.Uintptr
}

// isNumeric reports whether a reflect kind is numeric.
func isNumeric(k reflect.Kind) bool {
	return reflect.Int <= k && k <= reflect.Complex128
}

// isBoolean reports whether a reflect kind is Bool.
func isBoolean(k reflect.Kind) bool {
	return k == reflect.Bool
}

// isOrdered reports whether t is ordered.
func isOrdered(t *typeInfo) bool {
	k := t.Type.Kind()
	return isNumeric(k) || k == reflect.String
}

// isMapIndexing reports whether the given expression has the form
//
//		m[key]
//
// where m is a map.
func (tc *typechecker) isMapIndexing(node ast.Node) bool {
	index, ok := node.(*ast.Index)
	if !ok {
		return false
	}
	expr := index.Expr
	exprKind := tc.checkExpr(expr).Type.Kind()
	return exprKind == reflect.Map
}

// operatorFromAssignmentType returns an operator type from an assignment type.
func operatorFromAssignmentType(assignmentType ast.AssignmentType) ast.OperatorType {
	switch assignmentType {
	case ast.AssignmentAddition, ast.AssignmentIncrement:
		return ast.OperatorAddition
	case ast.AssignmentSubtraction, ast.AssignmentDecrement:
		return ast.OperatorSubtraction
	case ast.AssignmentMultiplication:
		return ast.OperatorMultiplication
	case ast.AssignmentDivision:
		return ast.OperatorDivision
	case ast.AssignmentModulo:
		return ast.OperatorModulo
	case ast.AssignmentAnd:
		return ast.OperatorBitAnd
	case ast.AssignmentOr:
		return ast.OperatorBitOr
	case ast.AssignmentXor:
		return ast.OperatorXor
	case ast.AssignmentAndNot:
		return ast.OperatorAndNot
	case ast.AssignmentLeftShift:
		return ast.OperatorLeftShift
	case ast.AssignmentRightShift:
		return ast.OperatorRightShift
	}
	panic("unexpected assignment type")
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

// nilOf returns a new type info representing a 'typed nil', that is the zero of
// type t.
func (tc *typechecker) nilOf(t reflect.Type) *typeInfo {
	switch t.Kind() {
	case reflect.Func:
		return &typeInfo{
			Properties: propertyHasValue | propertyIsNative,
			Type:       t,
			value:      tc.types.Zero(t),
		}
	case reflect.Interface:
		return &typeInfo{
			Properties: propertyHasValue,
			Type:       t,
			value:      nil,
		}
	default:
		return &typeInfo{
			Properties: propertyHasValue,
			Type:       t,
			value:      tc.types.Zero(t).Interface(),
		}
	}

}

// typedValue returns a constant type info value represented with a given
// type.
func (tc *typechecker) typedValue(ti *typeInfo, t reflect.Type) interface{} {
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
		case reflect.Uintptr:
			return uintptr(c.uint64())
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
	nv := tc.types.New(t).Elem()
	switch k {
	case reflect.Bool:
		nv.SetBool(c.bool())
	case reflect.String:
		nv.SetString(c.string())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		nv.SetInt(c.int64())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
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

// errTypeAssertion is called when the type typ does not implement the
// interface iface. It returns the corresponding compile-time error.
func (tc *typechecker) errTypeAssertion(typ reflect.Type, iface reflect.Type) error {
	msg := fmt.Sprintf("impossible type assertion:\n\t%s does not implement %s", typ, iface)
	num := iface.NumMethod()
	for i := 0; i < num; i++ {
		mi := iface.Method(i)
		mt, ok := typ.MethodByName(mi.Name)
		if !ok {
			ptr := tc.types.PtrTo(typ)
			_, ok = ptr.MethodByName(mi.Name)
			if ok {
				return fmt.Errorf("%s (%s method has pointer receiver)", msg, mi.Name)
			}
			return fmt.Errorf("%s (missing %s method)", msg, mi.Name)
		}
		numIn := mt.Type.NumIn() - 1
		numOut := mt.Type.NumOut()
		isVariadic := mt.Type.IsVariadic()
		sameParameters := mi.Type.NumIn() == numIn && mi.Type.NumOut() == numOut && mi.Type.IsVariadic() == isVariadic
		if sameParameters {
			for j := 0; j < numIn; j++ {
				if mi.Type.In(j) != mt.Type.In(j+1) {
					sameParameters = false
					break
				}
			}
		}
		if sameParameters {
			for j := 0; j < numOut; j++ {
				if mi.Type.Out(j) != mt.Type.Out(j) {
					sameParameters = false
					break
				}
			}
		}
		if !sameParameters {
			have := mt.Type.String()
			p := strings.IndexAny(have, " )")
			if have[p] == ' ' {
				p++
			}
			have = "func(" + have[p:]
			want := mi.Type.String()
			return fmt.Errorf("%s (wrong type for %s method)\n\t\thave %s\n\t\twant %s", msg, mi.Name, have, want)
		}
	}
	panic("unexpected")
}

// isValidIdentifier reports whether name is a valid identifier in the
// modality mod.
func isValidIdentifier(name string, mod checkingMod) bool {
	if name == "" || name == "_" {
		return false
	}
	for i, r := range name {
		if r != '_' && !unicode.IsLetter(r) && (i == 0 || !unicode.IsDigit(r)) {
			return false
		}
	}
	switch name {
	case
		"break",
		"case",
		"chan",
		"const",
		"continue",
		"default",
		"defer",
		"else",
		"fallthrough",
		"for",
		"func",
		"go",
		"goto",
		"if",
		"import",
		"interface",
		"map",
		"package",
		"range",
		"return",
		"struct",
		"select",
		"switch",
		"type",
		"var":
		return false
	case
		"end",
		"extends",
		"in",
		"macro",
		"raw",
		"render",
		"show",
		"using":
		return mod != templateMod
	case
		"and",
		"contains",
		"or",
		"not":
		return mod == programMod
	}
	return true
}
