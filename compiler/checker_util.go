// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"unicode"
	"unicode/utf8"

	"scriggo/compiler/ast"
	"scriggo/runtime"
)

var errTypeConversion = errors.New("failed type conversion")

type nilConvertionError struct {
	typ reflect.Type
}

func (err nilConvertionError) Error() string {
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

var boolOperators = [21]bool{
	ast.OperatorEqual:    true,
	ast.OperatorNotEqual: true,
	ast.OperatorAnd:      true,
	ast.OperatorOr:       true,
}

var intOperators = [21]bool{
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

var floatOperators = [21]bool{
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

var complexOperators = [21]bool{
	ast.OperatorEqual:          true,
	ast.OperatorNotEqual:       true,
	ast.OperatorAddition:       true,
	ast.OperatorSubtraction:    true,
	ast.OperatorMultiplication: true,
	ast.OperatorDivision:       true,
}

var stringOperators = [21]bool{
	ast.OperatorEqual:        true,
	ast.OperatorNotEqual:     true,
	ast.OperatorLess:         true,
	ast.OperatorLessEqual:    true,
	ast.OperatorGreater:      true,
	ast.OperatorGreaterEqual: true,
	ast.OperatorAddition:     true,
}

var interfaceOperators = [21]bool{
	ast.OperatorEqual:    true,
	ast.OperatorNotEqual: true,
}

var evalToBoolOperators = [21]bool{
	ast.OperatorEqual:        true,
	ast.OperatorNotEqual:     true,
	ast.OperatorLess:         true,
	ast.OperatorLessEqual:    true,
	ast.OperatorGreater:      true,
	ast.OperatorGreaterEqual: true,
	ast.OperatorAnd:          true,
	ast.OperatorOr:           true,
}

var operatorsOfKind = [...][21]bool{
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

type HTMLStringer interface{ HTML() string }
type CSSStringer interface{ CSS() string }
type JavaScriptStringer interface{ JavaScript() string }

var stringerType = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
var htmlStringerType = reflect.TypeOf((*HTMLStringer)(nil)).Elem()
var cssStringerType = reflect.TypeOf((*CSSStringer)(nil)).Elem()
var javaScriptStringerType = reflect.TypeOf((*JavaScriptStringer)(nil)).Elem()

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
		return nil, nilConvertionError{t2}

	case ti.IsConstant():
		// untyped constant.
		if k2 == reflect.Interface {
			if !tc.emptyMethodSet(t2) {
				return nil, errTypeConversion
			}
			_, err := ti.Constant.representedBy(ti.Type)
			return nil, err
		}
		return ti.Constant.representedBy(t2)

	case ti.Type == boolType:
		// untyped boolean value.
		if k2 == reflect.Bool || (k2 == reflect.Interface && tc.emptyMethodSet(t2)) {
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
			if !tc.emptyMethodSet(t2) {
				return nil, errTypeConversion
			}
			typ = ti.Type
		}
		tc.typeInfos[expr] = &typeInfo{Type: typ}

		switch expr := expr.(type) {

		case *ast.UnaryOperator:
			return tc.convert(tc.typeInfos[expr.Expr], expr.Expr, typ)

		case *ast.BinaryOperator:
			if op := expr.Operator(); op == ast.OperatorLeftShift || op == ast.OperatorRightShift {
				_, err := tc.convert(tc.typeInfos[expr.Expr1], expr.Expr1, typ)
				if err != nil {
					return nil, err
				}
				if k := typ.Kind(); k < reflect.Int || k > reflect.Uintptr {
					return nil, fmt.Errorf("invalid operation: %s (shift of type %s)", expr, typ)
				}
				return nil, nil
			}
			_, err := tc.convert(tc.typeInfos[expr.Expr1], expr.Expr1, typ)
			if err != nil {
				return nil, err
			}
			return tc.convert(tc.typeInfos[expr.Expr2], expr.Expr2, typ)

		default:
			panic(fmt.Errorf("BUG: unexpected expr %s (type %T) with type info %s", expr, expr, ti))

		}

	}

}

// declaredInThisBlock reports whether name is declared in this block.
func (tc *typechecker) declaredInThisBlock(name string) bool {
	_, ok := tc.lookupScopesElem(name, true)
	return ok
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
	case "delete":
		fun = func(m interface{}, key interface{}) {
			reflect.ValueOf(m).SetMapIndex(reflect.ValueOf(key), reflect.Value{})
		}
	case "panic":
		fun = func(env *runtime.Env, v interface{}) {
			if env.Exited() {
				return
			}
			panic(v)
		}
	case "print":
		fun = func(env *runtime.Env, args ...interface{}) {
			env.Print(args...)
		}
	case "println":
		fun = func(env *runtime.Env, args ...interface{}) {
			env.Println(args...)
		}
	case "recover":
		// This predefined function should only be used with the 'go'
		// statement and not the 'defer' statement.
		fun = func() {}
	}
	rv := reflect.ValueOf(fun)
	return &typeInfo{
		Properties: propertyHasValue | propertyIsPredefined,
		Type:       removeEnvArg(rv.Type(), false),
		value:      rv,
	}
}

// emptyMethodSet reports whether the given interface as an empty method set.
func (tc *typechecker) emptyMethodSet(interf reflect.Type) bool {
	return interf == emptyInterfaceType || tc.types.ConvertibleTo(intType, interf)
}

// fieldByName returns the struct field with the given name and a boolean
// indicating if the field was found.
//
// If name is unexported and the type is predefined, name is transformed and the
// new name is returned. For further information about this check the
// documentation of the type checking of an *ast.StructType.
func (tc *typechecker) fieldByName(t *typeInfo, name string) (*typeInfo, string, bool) {
	newName := name
	firstChar, _ := utf8.DecodeRuneInString(name)
	if !t.IsPredefined() && !unicode.Is(unicode.Lu, firstChar) {
		name = "𝗽" + strconv.Itoa(tc.compilation.UniqueIndex(tc.path)) + name
		newName = name
	}
	if t.Type.Kind() == reflect.Struct {
		field, ok := t.Type.FieldByName(name)
		if ok {
			return &typeInfo{Type: field.Type, Properties: t.Properties & propertyAddressable}, newName, true
		}
	}
	if t.Type.Kind() == reflect.Ptr {
		field, ok := t.Type.Elem().FieldByName(name)
		if ok {
			return &typeInfo{Type: field.Type, Properties: propertyAddressable}, newName, true
		}
	}
	return nil, newName, false
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

// addMissingTypes adds the type to the function parameters that do not have
// one.
// For example:
//
//      func(a, b int) (c, d string)
//
// is transformed into:
//
//      func(a int, b int) (c string, d string)
//
func (tc *typechecker) addMissingTypes(funType *ast.FuncType) {
	for _, params := range [2][]*ast.Parameter{funType.Parameters, funType.Result} {
		if len(params) == 0 {
			continue
		}
		typ := params[len(params)-1].Type
		for i := len(params) - 1; i >= 0; i-- {
			if params[i].Type != nil {
				typ = params[i].Type
			}
			params[i].Type = typ
		}
	}
}

type invalidTypeInAssignment string

func (err invalidTypeInAssignment) Error() string {
	return string(err)
}

func newInvalidTypeInAssignment(x *typeInfo, expr ast.Expression, t reflect.Type) invalidTypeInAssignment {
	return invalidTypeInAssignment(fmt.Sprintf("cannot use %s (type %s) as type %s", expr, x.Type, t))
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
	if x.Type == t {
		return nil
	}
	if t.Kind() == reflect.Interface {
		if !tc.types.Implements(x.Type, t) {
			return newInvalidTypeInAssignment(x, expr, t)
		}
		return nil
	}
	if !tc.types.AssignableTo(x.Type, t) {
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

// isComparison reports whether op is a comparison operator.
func isComparison(op ast.OperatorType) bool {
	return op >= ast.OperatorEqual && op <= ast.OperatorGreaterEqual
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

// isSelectorOfMapIndexing reports whether the given expression has the form
//
//		m[key].field
//
// where m is a map.
func (tc *typechecker) isSelectorOfMapIndexing(expr ast.Expression) bool {
	selector, ok := expr.(*ast.Selector)
	if !ok {
		return false
	}
	return tc.isMapIndexing(selector.Expr)
}

// macroToFunc converts a macro node into a function node.
func macroToFunc(macro *ast.Macro) *ast.Func {
	pos := macro.Pos()
	body := ast.NewBlock(pos, macro.Body)
	return ast.NewFunc(pos, macro.Ident, macro.Type, body)
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
func (tc *typechecker) methodByName(t *typeInfo, name string) (*typeInfo, receiverTransformation, bool) {

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
				ti := &typeInfo{
					Type:       removeEnvArg(methExpr.Type(), false),
					value:      methExpr,
					Properties: propertyIsPredefined | propertyHasValue,
				}
				return ti, receiverNoTransform, true
			}
		}

		// Method expression on concrete type.
		if method, ok := t.Type.MethodByName(name); ok {
			return &typeInfo{
				Type:       removeEnvArg(method.Type, true),
				value:      method.Func,
				Properties: propertyIsPredefined | propertyHasValue,
			}, receiverNoTransform, true
		}
		return nil, receiverNoTransform, false
	}

	// Method calls and method values on interfaces.
	if t.Type.Kind() == reflect.Interface {
		if method, ok := t.Type.MethodByName(name); ok {
			ti := &typeInfo{
				Type:       removeEnvArg(method.Type, true),
				value:      name,
				MethodType: methodValueInterface,
				Properties: propertyIsPredefined | propertyHasValue,
			}
			return ti, receiverNoTransform, true
		}
		return nil, receiverNoTransform, false
	}

	// Method calls and method values on concrete types.
	method := tc.types.Zero(t.Type).MethodByName(name)
	methodExplicitRcvr, _ := t.Type.MethodByName(name)
	if method.IsValid() {
		ti := &typeInfo{
			Type:       removeEnvArg(method.Type(), false),
			value:      methodExplicitRcvr.Func,
			Properties: propertyIsPredefined | propertyHasValue,
			MethodType: methodValueConcrete,
		}
		// Check if pointer is defined on T or *T when called on a *T receiver.
		if t.Type.Kind() == reflect.Ptr {
			// TODO(Gianluca): always check first if t has method m, regardless
			// of whether it's a pointer receiver or not. If m exists, it's
			// done. Else, if receiver was a pointer, call MethodByName on t
			// (which is a pointer).
			if method, methodOnT := t.Type.Elem().MethodByName(name); methodOnT {
				// Needs indirection: x.m -> (*x).m
				ti.Type = removeEnvArg(reflect.Zero(t.Type).MethodByName(name).Type(), false)
				ti.value = method.Func
				ti.Properties |= propertyHasValue
				return ti, receiverAddIndirect, true
			}
		}
		return ti, receiverNoTransform, true
	}
	if t.Type.Kind() != reflect.Ptr {
		method = tc.types.Zero(tc.types.PtrTo(t.Type)).MethodByName(name)
		methodExplicitRcvr, _ := tc.types.PtrTo(t.Type).MethodByName(name)
		if method.IsValid() {
			return &typeInfo{
				Type:       removeEnvArg(method.Type(), false),
				value:      methodExplicitRcvr.Func,
				Properties: propertyIsPredefined | propertyHasValue,
				MethodType: methodValueConcrete,
			}, receiverAddAddress, true
		}
	}
	return nil, receiverNoTransform, false
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

// printedAsJavaScript reports whether a type can be printed as JavaScript.
// It returns an error it the type cannot be printed.
func printedAsJavaScript(t reflect.Type) error {
	kind := t.Kind()
	if reflect.Bool <= kind && kind <= reflect.Float64 || kind == reflect.String ||
		t.Implements(javaScriptStringerType) {
		return nil
	}
	switch kind {
	case reflect.Array:
		if err := printedAsJavaScript(t.Elem()); err != nil {
			return fmt.Errorf("array of %s cannot be printed as JavaScript", t.Elem())
		}
	case reflect.Interface:
	case reflect.Map:
		key := t.Key().Kind()
		switch {
		case key == reflect.String:
		case reflect.Bool <= key && key <= reflect.Complex128:
		case t.Implements(stringerType):
		default:
			return fmt.Errorf("map with %s key cannot be printed as JavaScript", t.Key())
		}
		err := printedAsJavaScript(t.Elem())
		if err != nil {
			return fmt.Errorf("map with %s element cannot be printed as JavaScript", t.Elem())
		}
	case reflect.Ptr, reflect.UnsafePointer:
		return printedAsJavaScript(t.Elem())
	case reflect.Slice:
		if err := printedAsJavaScript(t.Elem()); err != nil {
			return fmt.Errorf("slice of %s cannot be printed as JavaScript", t.Elem())
		}
	case reflect.Struct:
		n := t.NumField()
		for i := 0; i < n; i++ {
			field := t.Field(i)
			if field.PkgPath == "" {
				if err := printedAsJavaScript(field.Type); err != nil {
					return fmt.Errorf("struct containing %s cannot be printed as JavaScript", field.Type)
				}
			}
		}
	default:
		return fmt.Errorf("type %s cannot be printed as JavaScript", t)
	}
	return nil
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
func representedBy(t1 *typeInfo, t2 reflect.Type) (constant, error) {
	if t1.IsConstant() {
		if t2.Kind() == reflect.Interface {
			if t1.Type.Implements(t2) {
				return nil, nil
			}
			return nil, fmt.Errorf("cannot convert %v (type %s) to type %s", t1.Constant, t1, t2)
		}
		c, err := t1.Constant.representedBy(t2)
		if err != nil {
			if err == errNotRepresentable {
				err = fmt.Errorf("cannot convert %v (type %s) to type %s", t1.Constant, t1, t2)
			}
			return nil, err
		}
		return c, nil
	}
	if t2.Kind() != reflect.Bool {
		return nil, fmt.Errorf("cannot use unsigned bool as type %s in assignment", t2)
	}
	return nil, nil
}

// nilOf returns a new type info representing a 'typed nil', that is the zero of
// type t.
func (tc *typechecker) nilOf(t reflect.Type) *typeInfo {
	switch t.Kind() {
	case reflect.Func:
		return &typeInfo{
			Properties: propertyHasValue | propertyIsPredefined,
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

// noSpacePosition returns the position of the text in n once the leading and
// trailing ASCII non-space bytes have been sliced off. If there are no bytes
// to slice off, it returns the position of n. If the text in n contains only
// ASCII space bytes, it returns nil.
func noSpacePosition(n *ast.Text) *ast.Position {
	// Search the first non white space byte.
	var i int
	for i = 0; i < len(n.Text); i++ {
		if !isASCIISpace(n.Text[i]) {
			break
		}
	}
	if i == len(n.Text) {
		return nil
	}
	// Search the last non white space byte.
	var j int
	for j = len(n.Text) - 1; j > i; j-- {
		if !isASCIISpace(n.Text[j]) {
			break
		}
	}
	if i == 0 && j == len(n.Text)-1 {
		return n.Pos()
	}
	// Build the position.
	pos := *n.Pos()
	pos.Start += i
	pos.End = pos.Start + j
	for _, c := range n.Text[:i] {
		if c == '\n' {
			pos.Line++
			pos.Column = 1
		} else {
			pos.Column++
		}
	}
	return &pos
}
