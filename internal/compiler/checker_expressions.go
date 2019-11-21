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

	"scriggo/ast"
	"scriggo/runtime"
)

type scopeElement struct {
	t    *TypeInfo
	decl *ast.Identifier
}

type typeCheckerScope map[string]scopeElement

var boolType = reflect.TypeOf(false)
var uintType = reflect.TypeOf(uint(0))
var uint8Type = reflect.TypeOf(uint8(0))
var int32Type = reflect.TypeOf(int32(0))
var byteSliceType = reflect.TypeOf([]byte(nil))

var uint8TypeInfo = &TypeInfo{Type: uint8Type, Properties: PropertyIsType | PropertyPredeclared}
var int32TypeInfo = &TypeInfo{Type: int32Type, Properties: PropertyIsType | PropertyPredeclared}

var untypedBoolTypeInfo = &TypeInfo{Type: boolType, Properties: PropertyUntyped}

var envType = reflect.TypeOf(&runtime.Env{})

var universe = typeCheckerScope{
	"append":     {t: &TypeInfo{Properties: PropertyPredeclared}},
	"cap":        {t: &TypeInfo{Properties: PropertyPredeclared}},
	"close":      {t: &TypeInfo{Properties: PropertyPredeclared}},
	"complex":    {t: &TypeInfo{Properties: PropertyPredeclared}},
	"copy":       {t: &TypeInfo{Properties: PropertyPredeclared}},
	"delete":     {t: &TypeInfo{Properties: PropertyPredeclared}},
	"imag":       {t: &TypeInfo{Properties: PropertyPredeclared}},
	"iota":       {t: &TypeInfo{Properties: PropertyPredeclared, Type: intType}},
	"len":        {t: &TypeInfo{Properties: PropertyPredeclared}},
	"make":       {t: &TypeInfo{Properties: PropertyPredeclared}},
	"new":        {t: &TypeInfo{Properties: PropertyPredeclared}},
	"nil":        {t: &TypeInfo{Properties: PropertyUntyped | PropertyPredeclared}},
	"panic":      {t: &TypeInfo{Properties: PropertyPredeclared}},
	"print":      {t: &TypeInfo{Properties: PropertyPredeclared}},
	"println":    {t: &TypeInfo{Properties: PropertyPredeclared}},
	"real":       {t: &TypeInfo{Properties: PropertyPredeclared}},
	"recover":    {t: &TypeInfo{Properties: PropertyPredeclared}},
	"byte":       {t: uint8TypeInfo},
	"bool":       {t: &TypeInfo{Type: boolType, Properties: PropertyIsType | PropertyPredeclared}},
	"complex128": {t: &TypeInfo{Type: complex128Type, Properties: PropertyIsType | PropertyPredeclared}},
	"complex64":  {t: &TypeInfo{Type: complex64Type, Properties: PropertyIsType | PropertyPredeclared}},
	"error":      {t: &TypeInfo{Type: reflect.TypeOf((*error)(nil)).Elem(), Properties: PropertyIsType | PropertyPredeclared}},
	"float32":    {t: &TypeInfo{Type: reflect.TypeOf(float32(0)), Properties: PropertyIsType | PropertyPredeclared}},
	"float64":    {t: &TypeInfo{Type: float64Type, Properties: PropertyIsType | PropertyPredeclared}},
	"false":      {t: &TypeInfo{Type: boolType, Properties: PropertyUntyped, Constant: boolConst(false)}},
	"int":        {t: &TypeInfo{Type: intType, Properties: PropertyIsType | PropertyPredeclared}},
	"int16":      {t: &TypeInfo{Type: reflect.TypeOf(int16(0)), Properties: PropertyIsType | PropertyPredeclared}},
	"int32":      {t: int32TypeInfo},
	"int64":      {t: &TypeInfo{Type: reflect.TypeOf(int64(0)), Properties: PropertyIsType | PropertyPredeclared}},
	"int8":       {t: &TypeInfo{Type: reflect.TypeOf(int8(0)), Properties: PropertyIsType | PropertyPredeclared}},
	"rune":       {t: int32TypeInfo},
	"string":     {t: &TypeInfo{Type: stringType, Properties: PropertyIsType | PropertyPredeclared}},
	"true":       {t: &TypeInfo{Type: boolType, Properties: PropertyUntyped, Constant: boolConst(true)}},
	"uint":       {t: &TypeInfo{Type: uintType, Properties: PropertyIsType | PropertyPredeclared}},
	"uint16":     {t: &TypeInfo{Type: reflect.TypeOf(uint16(0)), Properties: PropertyIsType | PropertyPredeclared}},
	"uint32":     {t: &TypeInfo{Type: reflect.TypeOf(uint32(0)), Properties: PropertyIsType | PropertyPredeclared}},
	"uint64":     {t: &TypeInfo{Type: reflect.TypeOf(uint64(0)), Properties: PropertyIsType | PropertyPredeclared}},
	"uint8":      {t: uint8TypeInfo},
	"uintptr":    {t: &TypeInfo{Type: reflect.TypeOf(uintptr(0)), Properties: PropertyIsType | PropertyPredeclared}},
}

type scopeVariable struct {
	ident      string
	scopeLevel int
	node       ast.Node
}

// showMacroIgnoredTi is the TypeInfo of a ShowMacro identifier which is
// undefined but has been marked as to be ignored or "todo".
var showMacroIgnoredTi = &TypeInfo{}

// checkIdentifier checks an identifier. If using, ident is marked as "used".
func (tc *typechecker) checkIdentifier(ident *ast.Identifier, using bool) *TypeInfo {

	// If ident is an upvar, add it as upvar for current function and for all
	// nested functions and update all indexes.
	if tc.isUpVar(ident.Name) {
		upvar := ast.Upvar{
			Declaration: tc.getDeclarationNode(ident.Name),
			Index:       -1,
		}
		for _, fn := range tc.getNestedFuncs(ident.Name) {
			add := true
			for i, uv := range fn.Upvars {
				if uv.Declaration == upvar.Declaration {
					upvar.Index = int16(i)
					add = false
					break
				}
			}
			if add {
				fn.Upvars = append(fn.Upvars, upvar)
				upvar.Index = int16(len(fn.Upvars) - 1)
			}
		}
	}

	// Handle predeclared identifier "iota".
	i, ok := tc.lookupScopes(ident.Name, false)
	if ok && i == universe["iota"].t && tc.iota >= 0 {
		return &TypeInfo{
			Constant:   int64Const(tc.iota),
			Type:       intType,
			Properties: PropertyUntyped,
		}
	}

	// If identifiers is a ShowMacro identifier, first needs to check if
	// ShowMacro contains a "or ignore" or "or todo" option. In such cases,
	// error should not be returned, and function call should be removed
	// from tree.
	if !ok {
		for _, sm := range tc.showMacros {
			if sm.Macro == ident {
				switch sm.Or {
				case ast.ShowMacroOrIgnore:
					return showMacroIgnoredTi
				case ast.ShowMacroOrTodo:
					if tc.opts.FailOnTODO {
						panic(tc.errorf(ident, "macro %s is not defined: must be implemented", ident.Name))
					}
					return showMacroIgnoredTi
				case ast.ShowMacroOrError:
					// Do not handle this identifier in a special way: just
					// return an 'undefined' error; this is the default
					// behaviour.
				}
			}
		}
		panic(tc.errorf(ident, "undefined: %s", ident.Name))
	}

	if i.IsBuiltinFunction() {
		panic(tc.errorf(ident, "use of builtin %s not in function call", ident.Name))
	}

	// Mark identifier as "used".
	if using {
		for i := len(tc.unusedVars) - 1; i >= 0; i-- {
			v := tc.unusedVars[i]
			if v.ident == ident.Name {
				v.node = nil
				break
			}
		}
	}

	// For "." imported packages, marks package as used.
unusedLoop:
	for pkg, decls := range tc.unusedImports {
		for _, d := range decls {
			if d != ident.Name {
				delete(tc.unusedImports, pkg)
				break unusedLoop
			}
		}
	}

	tc.typeInfos[ident] = i
	return i
}

// checkArrayType checks an array type. If the array node is used in a composite
// literal and has no length (has ellipses), the length parameters is used
// instead.
func (tc *typechecker) checkArrayType(array *ast.ArrayType, length int) *TypeInfo {
	elem := tc.checkType(array.ElementType)
	if array.Len == nil { // ellipsis.
		if length == -1 {
			panic(tc.errorf(array, "use of [...] array outside of array literal"))
		}
		tc.typeInfos[array] = &TypeInfo{Properties: PropertyIsType, Type: tc.types.ArrayOf(length, elem.Type)}
		return tc.typeInfos[array]
	}
	len := tc.checkExpr(array.Len)
	if !len.IsConstant() {
		panic(tc.errorf(array, "non-constant array bound %s", array.Len))
	}
	c, err := tc.convertExplicitly(len, intType)
	if err != nil {
		panic(tc.errorf(array, "%s", err))
	}
	b := int(c.int64())
	if b < 0 {
		panic(tc.errorf(array, "array bound must be non-negative"))
	}
	if b < length {
		panic(tc.errorf(array, "array index %d out of bounds [0:%d]", length-1, b))
	}
	tc.typeInfos[array] = &TypeInfo{Properties: PropertyIsType, Type: tc.types.ArrayOf(b, elem.Type)}
	return tc.typeInfos[array]
}

// checkExpr type checks an expression and returns its type info.
func (tc *typechecker) checkExpr(expr ast.Expression) *TypeInfo {
	ti := tc.typeof(expr, false)
	if ti.IsType() {
		panic(tc.errorf(expr, "type %s is not an expression", ti))
	}
	tc.typeInfos[expr] = ti
	if ti.UntypedNonConstantInteger() {
		tc.untypedIntegerRoot = expr
	}
	return ti
}

// checkType type checks a type and returns its type info.
func (tc *typechecker) checkType(expr ast.Expression) *TypeInfo {
	ti := tc.typeof(expr, true)
	if !ti.IsType() {
		panic(tc.errorf(expr, "%s is not a type", expr))
	}
	tc.typeInfos[expr] = ti
	return ti
}

// checkExprOrType type checks an expression or a type and returns its type
// info.
func (tc *typechecker) checkExprOrType(expr ast.Expression) *TypeInfo {
	ti := tc.typeof(expr, true)
	tc.typeInfos[expr] = ti
	return ti
}

// typeof type checks an expression or type and returns its type info.
// typeExpected reports whether a type is expected; it only affects error
// messages.
//
// typeof should not be called directly, use checkExpr, checkType or
// checkExprOrType.
func (tc *typechecker) typeof(expr ast.Expression, typeExpected bool) *TypeInfo {

	if isBlankIdentifier(expr) {
		panic(tc.errorf(expr, "cannot use _ as value"))
	}

	ti := tc.typeInfos[expr]
	if ti != nil {
		return ti
	}

	switch expr := expr.(type) {

	case *ast.BasicLiteral:
		var typ reflect.Type
		switch expr.Type {
		case ast.StringLiteral:
			typ = stringType
		case ast.RuneLiteral:
			typ = runeType
		case ast.IntLiteral:
			typ = intType
		case ast.FloatLiteral:
			typ = float64Type
		case ast.ImaginaryLiteral:
			typ = complex128Type
		}
		c, err := parseBasicLiteral(expr.Type, expr.Value)
		if err != nil {
			panic(tc.errorf(expr, err.Error()))
		}
		return &TypeInfo{
			Type:       typ,
			Properties: PropertyUntyped,
			Constant:   c,
		}

	case *ast.UnaryOperator:
		t := tc.checkExprOrType(expr.Expr)
		if t.IsType() {
			if expr.Op == ast.OperatorMultiplication {
				return &TypeInfo{Properties: PropertyIsType, Type: tc.types.PtrTo(t.Type)}
			}
			panic(tc.errorf(expr, "type %s is not an expression", t))
		}
		ti := &TypeInfo{
			Type:       t.Type,
			Properties: t.Properties & PropertyUntyped,
		}
		var k reflect.Kind
		if !t.Nil() {
			k = t.Type.Kind()
		}
		switch expr.Op {
		case ast.OperatorNot:
			if t.Nil() || k != reflect.Bool {
				panic(tc.errorf(expr, "invalid operation: ! %s", t))
			}
			if t.IsConstant() {
				ti.Constant, _ = t.Constant.unaryOp(ast.OperatorNot)
			}
		case ast.OperatorAddition:
			if t.Nil() || !isNumeric(k) {
				panic(tc.errorf(expr, "invalid operation: + %s", t))
			}
			if t.IsConstant() {
				ti.Constant = t.Constant
			}
		case ast.OperatorSubtraction:
			if t.Nil() || !isNumeric(k) {
				panic(tc.errorf(expr, "invalid operation: - %s", t))
			}
			if t.IsConstant() {
				ti.Constant, _ = t.Constant.unaryOp(ast.OperatorSubtraction)
			}
		case ast.OperatorMultiplication:
			if t.Nil() {
				panic(tc.errorf(expr, "invalid indirect of nil"))
			}
			if k != reflect.Ptr {
				if typeExpected {
					panic(tc.errorf(expr, "%s is not a type", expr))
				}
				panic(tc.errorf(expr, "invalid indirect of %s (type %s)", expr.Expr, t))
			}
			ti.Type = t.Type.Elem()
			ti.Properties = ti.Properties | PropertyAddressable
		case ast.OperatorAnd:
			if _, ok := expr.Expr.(*ast.CompositeLiteral); !ok && !t.Addressable() {
				panic(tc.errorf(expr, "cannot take the address of %s", expr.Expr))
			}
			ti.Type = tc.types.PtrTo(t.Type)
			// When taking the address of a variable, such variable must be
			// marked as "indirect".
			if ident, ok := expr.Expr.(*ast.Identifier); ok {
			scopesLoop:
				for i := len(tc.scopes) - 1; i >= 0; i-- {
					for n := range tc.scopes[i] {
						if n == ident.Name {
							tc.indirectVars[tc.scopes[i][n].decl] = true
							break scopesLoop
						}
					}
				}
			}
		case ast.OperatorXor:
			if t.Nil() || !isInteger(k) {
				panic(tc.errorf(expr, "invalid operation: ^ %s", t))
			}
			if t.IsConstant() {
				ti.Constant, _ = t.Constant.unaryOp(ast.OperatorXor)
			}
		case ast.OperatorReceive:
			if t.Nil() {
				panic(tc.errorf(expr, "use of untyped nil"))
			}
			if k != reflect.Chan {
				panic(tc.errorf(expr, "invalid operation: %s (receive from non-chan type %s)", expr, t.Type))
			}
			if t.Type.ChanDir() == reflect.SendDir {
				// Expression <-make(...) is printed as <-(make(...)) in the error message.
				var s string
				if call, ok := expr.Expr.(*ast.Call); ok && tc.typeInfos[call.Func].IsBuiltinFunction() {
					s = expr.Op.String() + "(" + expr.Expr.String() + ")"
				} else {
					s = expr.String()
				}
				panic(tc.errorf(expr, "invalid operation: %s (receive from send-only type %s)", s, t.Type))
			}
			ti.Type = t.Type.Elem()
		}
		return ti

	case *ast.BinaryOperator:
		t, err := tc.binaryOp(expr.Expr1, expr.Op, expr.Expr2)
		if err != nil {
			panic(tc.errorf(expr, "invalid operation: %v (%s)", expr, err))
		}
		return t

	case *ast.Identifier:
		t := tc.checkIdentifier(expr, true)
		if t.IsPackage() {
			panic(tc.errorf(expr, "use of package %s without selector", expr))
		}
		return t

	case *ast.StructType:
		fields := []reflect.StructField{}
		for _, fd := range expr.Fields {
			typ := tc.checkType(fd.Type).Type
			if fd.IdentifierList == nil {
				// Not implemented: see https://github.com/open2b/scriggo/issues/367
			} else {
				// Explicit field declaration.
				for _, ident := range fd.IdentifierList {
					// If the field name is unexported, it's impossible to
					// create an new reflect.Type due to the limits that the
					// package 'reflect' currently has. The solution adopted is
					// to prepone an unicode character 𝗽 which is considered
					// unexported by the specifications of Go but is allowed by
					// the reflect.
					//
					// In addition to this, two values with type struct declared
					// in two different packages cannot be compared (types are
					// different) because the package paths are different; the
					// reflect package does not have the ability to set the such
					// path; to work around the problem, an identifier is put in
					// the middle of the character 𝗽 and the original field
					// name; this makes the field unique to a given package,
					// resulting in the unability of make comparisons with that
					// types.
					name := ident.Name
					if fc, _ := utf8.DecodeRuneInString(name); !unicode.Is(unicode.Lu, fc) {
						name = "𝗽" + strconv.Itoa(tc.currentPkgIndex()) + ident.Name
					}
					fields = append(fields, reflect.StructField{
						Name:      name,
						Type:      typ,
						Anonymous: false,
					})
				}
			}
		}
		t := tc.types.StructOf(fields)
		return &TypeInfo{
			Type:       t,
			Properties: PropertyIsType,
		}

	case *ast.MapType:
		key := tc.checkType(expr.KeyType)
		value := tc.checkType(expr.ValueType)
		defer func() {
			if rec := recover(); rec != nil {
				panic(tc.errorf(expr, "invalid map key type %s", key))
			}
		}()
		return &TypeInfo{Properties: PropertyIsType, Type: tc.types.MapOf(key.Type, value.Type)}

	case *ast.SliceType:
		elem := tc.checkType(expr.ElementType)
		return &TypeInfo{Properties: PropertyIsType, Type: tc.types.SliceOf(elem.Type)}

	case *ast.ArrayType:
		return tc.checkArrayType(expr, -1)

	case *ast.ChanType:
		var dir reflect.ChanDir
		switch expr.Direction {
		case ast.NoDirection:
			dir = reflect.BothDir
		case ast.ReceiveDirection:
			dir = reflect.RecvDir
		case ast.SendDirection:
			dir = reflect.SendDir
		}
		elem := tc.checkType(expr.ElementType)
		return &TypeInfo{Properties: PropertyIsType, Type: tc.types.ChanOf(dir, elem.Type)}

	case *ast.CompositeLiteral:
		return tc.checkCompositeLiteral(expr, nil)

	case *ast.Interface:
		return &TypeInfo{Type: emptyInterfaceType, Properties: PropertyIsType | PropertyPredeclared}

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
				t := tc.checkType(param.Type)
				if variadic && i == numIn-1 {
					in[i] = tc.types.SliceOf(t.Type)
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
			if res.Type == nil {
				out[i] = out[i+1]
			} else {
				c := tc.checkType(res.Type)
				out[i] = c.Type
			}
		}
		expr.Reflect = tc.types.FuncOf(in, out, variadic)
		return &TypeInfo{Type: expr.Reflect, Properties: PropertyIsType}

	case *ast.Func:
		tc.enterScope()
		t := tc.checkType(expr.Type)
		expr.Type.Reflect = t.Type
		tc.addToAncestors(expr)
		// Adds parameters to the function body scope.
		fillParametersTypes(expr.Type.Parameters)
		isVariadic := expr.Type.IsVariadic
		for i, f := range expr.Type.Parameters {
			t := tc.checkType(f.Type)
			if f.Ident != nil {
				if isVariadic && i == len(expr.Type.Parameters)-1 {
					tc.assignScope(f.Ident.Name, &TypeInfo{Type: tc.types.SliceOf(t.Type), Properties: PropertyAddressable}, nil)
					continue
				}
				tc.assignScope(f.Ident.Name, &TypeInfo{Type: t.Type, Properties: PropertyAddressable}, nil)
			}
		}
		// Adds named return values to the function body scope.
		fillParametersTypes(expr.Type.Result)
		for _, f := range expr.Type.Result {
			t := tc.checkType(f.Type)
			if f.Ident != nil {
				tc.assignScope(f.Ident.Name, &TypeInfo{Type: t.Type, Properties: PropertyAddressable}, nil)
			}
		}
		tc.checkNodes(expr.Body.Nodes)
		// «If the function's signature declares result parameters, the
		// function body's statement list must end in a terminating
		// statement.»
		if len(expr.Type.Result) > 0 {
			if !tc.terminating {
				panic(tc.errorf(expr, "missing return at end of function"))
			}
		}
		tc.ancestors = tc.ancestors[:len(tc.ancestors)-1]
		tc.exitScope()
		return &TypeInfo{Type: t.Type}

	case *ast.Call:
		types, _, _ := tc.checkCallExpression(expr, false)
		if len(types) == 0 {
			panic(tc.errorf(expr, "%v used as value", expr))
		}
		if len(types) > 1 {
			panic(tc.errorf(expr, "multiple-value %v in single-value context", expr))
		}
		return types[0]

	case *ast.Index:
		t := tc.checkExpr(expr.Expr)
		if t.Nil() {
			panic(tc.errorf(expr, "use of untyped nil"))
		}
		t.setValue(nil)
		kind := t.Type.Kind()
		switch kind {
		case reflect.String:
			tc.checkIndex(expr.Index, t, false)
			return &TypeInfo{Type: uint8Type}
		case reflect.Slice:
			tc.checkIndex(expr.Index, t, false)
			return &TypeInfo{Type: t.Type.Elem(), Properties: PropertyAddressable}
		case reflect.Array:
			tc.checkIndex(expr.Index, t, false)
			ti := &TypeInfo{Type: t.Type.Elem()}
			if t.Addressable() {
				ti.Properties = PropertyAddressable
			}
			return ti
		case reflect.Ptr:
			elemType := t.Type.Elem()
			if elemType.Kind() != reflect.Array {
				panic(tc.errorf(expr, "invalid operation: %v (type %s does not support indexing)", expr, t))
			}
			tc.checkIndex(expr.Index, t, false)
			// Transform pa[i] to (*pa)[i].
			unOp := ast.NewUnaryOperator(expr.Expr.Pos(), ast.OperatorMultiplication, expr.Expr)
			tc.typeInfos[unOp] = &TypeInfo{Type: elemType}
			expr.Expr = unOp
			return &TypeInfo{Type: elemType.Elem(), Properties: PropertyAddressable}
		case reflect.Map:
			key := tc.checkExpr(expr.Index)
			if err := tc.isAssignableTo(key, expr.Index, t.Type.Key()); err != nil {
				if _, ok := err.(invalidTypeInAssignment); ok {
					panic(tc.errorf(expr, "%s in map index", err))
				}
				panic(tc.errorf(expr, "%s", err))
			}
			if key.Nil() {
				key = tc.nilOf(t.Type.Key())
				tc.typeInfos[expr.Index] = key
			} else {
				key.setValue(t.Type.Key())
			}
			return &TypeInfo{Type: t.Type.Elem()}
		default:
			panic(tc.errorf(expr, "invalid operation: %s (type %s does not support indexing)", expr, t.ShortString()))
		}

	case *ast.Slicing:
		t := tc.checkExpr(expr.Expr)
		if t.Nil() {
			panic(tc.errorf(expr, "use of untyped nil"))
		}
		t.setValue(nil)
		kind := t.Type.Kind()
		realType := t.Type
		realKind := kind
		switch kind {
		case reflect.String:
			if expr.IsFull {
				panic(tc.errorf(expr, "invalid operation %s (3-index slice of string)", expr))
			}
		case reflect.Slice:
		case reflect.Array:
			if !t.Addressable() {
				panic(tc.errorf(expr, "invalid operation %s (slice of unaddressable value)", expr))
			}
		default:
			if kind == reflect.Ptr {
				realType = t.Type.Elem()
				realKind = realType.Kind()
			}
			if realKind != reflect.Array {
				panic(tc.errorf(expr, "cannot slice %s (type %s)", expr.Expr, t.ShortString()))
			}
		}
		var lv, hv, mv constant
		if expr.Low != nil {
			lv = tc.checkIndex(expr.Low, t, true)
		}
		if expr.High != nil {
			hv = tc.checkIndex(expr.High, t, true)
		} else if expr.IsFull {
			panic(tc.errorf(expr, "middle index required in 3-index slice"))
		}
		if expr.Max != nil {
			mv = tc.checkIndex(expr.Max, t, true)
		} else if expr.IsFull {
			panic(tc.errorf(expr, "final index required in 3-index slice"))
		}
		if lv != nil && hv != nil && lv.int64() > hv.int64() {
			panic(tc.errorf(expr, "invalid slice index: %d > %d", lv, hv))
		}
		if lv != nil && mv != nil && lv.int64() > mv.int64() {
			panic(tc.errorf(expr, "invalid slice index: %d > %d", lv, mv))
		}
		if hv != nil && mv != nil && hv.int64() > mv.int64() {
			panic(tc.errorf(expr, "invalid slice index: %d > %d", hv, mv))
		}
		// Transform the tree: if a is a pointer to an array, a[low : high] is
		// shorthand for (*a)[low : high]; also, if a is a pointer to an array,
		// a[low : high : max] is shorthand for (*a)[low : high : max].
		if t.Type.Kind() == reflect.Ptr && t.Type.Elem().Kind() == reflect.Array {
			unOp := ast.NewUnaryOperator(expr.Expr.Pos(), ast.OperatorMultiplication, expr.Expr)
			tc.typeInfos[unOp] = &TypeInfo{
				Type: t.Type.Elem(),
			}
			expr.Expr = unOp
		}
		switch kind {
		case reflect.String, reflect.Slice:
			return &TypeInfo{Type: t.Type}
		case reflect.Array, reflect.Ptr:
			return &TypeInfo{Type: tc.types.SliceOf(realType.Elem())}
		}

	case *ast.Selector:
		// Package selector.
		if ident, ok := expr.Expr.(*ast.Identifier); ok {
			ti, ok := tc.lookupScopes(ident.Name, false)
			if ok {
				if ti.IsPackage() {
					delete(tc.unusedImports, ident.Name)
					if !unicode.Is(unicode.Lu, []rune(expr.Ident)[0]) {
						panic(tc.errorf(expr, "cannot refer to unexported name %s", expr))
					}
					pkg := ti.value.(*PackageInfo)
					v, ok := pkg.Declarations[expr.Ident]
					if !ok {
						// If identifiers is a ShowMacro identifier, first needs to check if
						// ShowMacro contains a "or ignore" or "or todo" option. In such cases,
						// error should not be returned, and function call should be removed
						// from tree.
						for _, sm := range tc.showMacros {
							if sm.Macro == ident {
								switch sm.Or {
								case ast.ShowMacroOrIgnore:
									return showMacroIgnoredTi
								case ast.ShowMacroOrTodo:
									if tc.opts.FailOnTODO {
										panic(tc.errorf(ident, "macro %s is not defined: must be implemented", ident.Name))
									}
									return showMacroIgnoredTi
								case ast.ShowMacroOrError:
									// Do not handle this identifier in a special way: just
									// return an 'undefined' error; this is the default
									// behaviour.
								}
							}
						}
						panic(tc.errorf(expr, "undefined: %v", expr))
					}
					// v is a predefined variable.
					if rv, ok := v.value.(*reflect.Value); v.Addressable() && ok {
						upvar := ast.Upvar{
							PredefinedName:  expr.Ident,
							PredefinedPkg:   ident.Name,
							PredefinedValue: rv,
							Index:           -1,
						}
						for _, fn := range tc.nestedFuncs() {
							add := true
							for i, uv := range fn.Upvars {
								if uv.PredefinedValue == upvar.PredefinedValue {
									upvar.Index = int16(i)
									add = false
									break
								}
							}
							if add {
								upvar.Index = int16(len(fn.Upvars) - 1)
								fn.Upvars = append(fn.Upvars, upvar)
							}
						}
					}
					tc.typeInfos[expr] = v
					return v
				}
			}
		}
		t := tc.checkExprOrType(expr.Expr)
		if t.IsType() {
			method, _, ok := tc.methodByName(t, expr.Ident)
			if !ok {
				panic(tc.errorf(expr, "%v undefined (type %s has no method %s)", expr, t, expr.Ident))
			}
			return method
		}
		method, trans, ok := tc.methodByName(t, expr.Ident)
		if ok {
			switch trans {
			case receiverAddAddress:
				if t.Addressable() {
					elem, _ := tc.lookupScopesElem(expr.Expr.(*ast.Identifier).Name, false)
					tc.indirectVars[elem.decl] = true
					expr.Expr = ast.NewUnaryOperator(expr.Pos(), ast.OperatorAnd, expr.Expr)
					tc.typeInfos[expr.Expr] = &TypeInfo{
						Type:       tc.types.PtrTo(t.Type),
						MethodType: t.MethodType,
					}
				}
			case receiverAddIndirect:
				expr.Expr = ast.NewUnaryOperator(expr.Pos(), ast.OperatorMultiplication, expr.Expr)
				tc.typeInfos[expr.Expr] = &TypeInfo{
					Type:       t.Type.Elem(),
					MethodType: t.MethodType,
				}
			}
			return method
		}
		field, newName, ok := tc.fieldByName(t, expr.Ident)
		if ok {
			expr.Ident = newName
			// Transform ps.F to (*ps).F, if ps is a defined pointer type and
			// (*ps).F is a valid selector expression denoting a field (but not
			// a method).
			if t.Type.Kind() == reflect.Ptr && t.Type.Elem().Kind() == reflect.Struct {
				unOp := ast.NewUnaryOperator(expr.Expr.Pos(), ast.OperatorMultiplication, expr.Expr)
				tc.typeInfos[unOp] = &TypeInfo{
					Type: t.Type.Elem(),
				}
				expr.Expr = unOp
			}
			return field
		}
		panic(tc.errorf(expr, "%v undefined (type %s has no field or method %s)", expr, t, expr.Ident))

	case *ast.TypeAssertion:
		t := tc.checkExpr(expr.Expr)
		if t.Nil() {
			panic(tc.errorf(expr, "use of untyped nil"))
		}
		if t.Type.Kind() != reflect.Interface {
			panic(tc.errorf(expr, "invalid type assertion: %v (non-interface type %s on left)", expr, t))
		}
		typ := tc.checkType(expr.Type)
		return &TypeInfo{
			Type:       typ.Type,
			Properties: t.Properties & PropertyAddressable,
		}

	}

	panic(fmt.Errorf("unexpected: %v (type %T)", expr, expr))
}

// checkIndex checks the type of expr as an index in a index or slice
// expression. If it is a constant returns the integer value, otherwise
// returns -1.
func (tc *typechecker) checkIndex(expr ast.Expression, t *TypeInfo, isSlice bool) constant {
	typ := t.Type
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	index := tc.checkExpr(expr)
	if index.Untyped() && !index.IsNumeric() || !index.Untyped() && !index.IsInteger() {
		if isSlice {
			panic(tc.errorf(expr, "invalid slice index %s (type %s)", expr, index))
		}
		panic(tc.errorf(expr, "non-integer %s index %s", typ.Kind(), expr))
	}
	index.setValue(intType)
	if index.IsConstant() {
		c, err := tc.convertExplicitly(index, intType)
		if err != nil {
			panic(tc.errorf(expr, "%s", err))
		}
		i := int(c.int64())
		if i < 0 {
			panic(tc.errorf(expr, "invalid %s index %s (index must be non-negative)", typ.Kind(), expr))
		}
		j := i
		if isSlice {
			j--
		}
		if t.IsConstant() {
			if s := t.Constant.string(); j >= len(s) {
				what := typ.Kind().String()
				if isSlice {
					what = "slice"
				}
				panic(tc.errorf(expr, "invalid %s index %s (out of bounds for %d-byte string)", what, expr, len(s)))
			}
		} else if typ.Kind() == reflect.Array && j > typ.Len() {
			panic(tc.errorf(expr, "invalid array index %s (out of bounds for %d-element array)", expr, typ.Len()))
		}
		return c
	}
	return nil
}

// binaryOp executes the binary expression t1 op t2 and returns its result.
// Returns an error if the operation can not be executed.
func (tc *typechecker) binaryOp(expr1 ast.Expression, op ast.OperatorType, expr2 ast.Expression) (*TypeInfo, error) {

	t1 := tc.checkExpr(expr1)
	t2 := tc.checkExpr(expr2)

	if t1.UntypedNonConstantInteger() && t2.UntypedNonConstantInteger() {
		// TODO: review:
		t1.Type = intType
		t2.Type = intType
	}

	if t1.UntypedNonConstantInteger() {
		// TODO: review:
		t1.Type = t2.Type
	}

	if t2.UntypedNonConstantInteger() {
		// TODO: review:
		t2.Type = t1.Type
	}

	if op == ast.OperatorLeftShift || op == ast.OperatorRightShift {
		if t2.Nil() {
			return nil, errors.New("cannot convert nil to type uint")
		}
		if t1.IsUntypedConstant() {
			if !t1.IsNumeric() {
				return nil, fmt.Errorf("shift of type %s", t1)
			}
		} else if !t1.IsInteger() {
			return nil, fmt.Errorf("shift of type %s", t1)
		}
		var c constant
		var err error
		if t2.IsConstant() {
			if !t2.IsNumeric() {
				return nil, fmt.Errorf("shift count type %s, must be unsigned integer", t2.ShortString())
			}
			if t1.IsConstant() {
				c, err = t1.Constant.binaryOp(op, t2.Constant)
				if err != nil {
					switch err {
					case errNegativeShiftCount:
						err = fmt.Errorf("invalid negative shift count: %s", t2.Constant)
					case errShiftCountTooLarge:
						err = fmt.Errorf("shift count too large: %s", t2.Constant)
					case errShiftCountTruncatedToInteger:
						err = fmt.Errorf("constant %s truncated to integer", t2.Constant)
					case errConstantOverflowUint:
						err = fmt.Errorf("constant %s overflows uint", t2.Constant)
					}
					return nil, err
				}
			} else {
				_, err = t2.Constant.representedBy(uintType)
				if err != nil {
					return nil, err
				}
				t2.setValue(uintType)
			}
		} else if !t2.IsInteger() {
			return nil, fmt.Errorf("shift count type %s, must be unsigned integer", t2.ShortString())
		}
		ti := &TypeInfo{Type: t1.Type}
		if t1.IsConstant() && t2.IsConstant() {
			ti.Constant = c
			if t1.Untyped() {
				ti.Properties = PropertyUntyped
			} else {
				ti.Constant, err = tc.convertExplicitly(ti, ti.Type)
				if err != nil {
					return nil, err
				}
			}
		} else if t1.IsUntypedConstant() {
			ti.Properties = PropertyUntyped
			tc.untypedIntegers[ti] = t1
		}
		t1.setValue(nil)
		t2.setValue(nil)
		return ti, nil
	}

	if t1.Nil() || t2.Nil() {
		if t1.Nil() && t2.Nil() {
			return nil, fmt.Errorf("operator %s not defined on nil", op)
		}
		if t1.Nil() {
			t1, t2 = t2, t1
		}
		_, err := tc.convertImplicitly(t2, t1.Type)
		if err != nil {
			return nil, err
		}
		if op != ast.OperatorEqual && op != ast.OperatorNotEqual {
			return nil, fmt.Errorf("operator %s not defined on %s", op, t1.Type.Kind())
		}
		return untypedBoolTypeInfo, nil
	}

	if t1.IsUntypedConstant() && t2.IsUntypedConstant() {
		k1 := t1.Type.Kind()
		k2 := t2.Type.Kind()
		if !(k1 == k2 || isNumeric(k1) && isNumeric(k2)) {
			return nil, fmt.Errorf("mismatched types %s and %s", t1.ShortString(), t2.ShortString())
		}
		c, err := t1.Constant.binaryOp(op, t2.Constant)
		if err != nil {
			if err == errInvalidOperation {
				switch op {
				case ast.OperatorModulo:
					if isComplex(t1.Type.Kind()) {
						return nil, errors.New("complex % operation")
					}
					return nil, errors.New("floating-point % operation")
				case ast.OperatorAnd, ast.OperatorOr, ast.OperatorXor:
					if isComplex(t1.Type.Kind()) {
						return nil, fmt.Errorf("operator %s not defined on untyped complex", op)
					}
					return nil, fmt.Errorf("operator %s not defined on untyped float", op)
				}
				return nil, fmt.Errorf("operator %s not defined on %s", op, t1.ShortString())
			}
			return nil, err
		}
		t := &TypeInfo{Constant: c, Properties: PropertyUntyped}
		if evalToBoolOperators[op] {
			t.Type = boolType
		} else {
			t.Type = t1.Type
			if k2 > k1 {
				t.Type = t2.Type
			}
		}
		return t, nil
	}

	if t1.Untyped() {
		c, err := tc.convertImplicitly(t1, t2.Type)
		if err != nil {
			if err == errNotRepresentable {
				err = fmt.Errorf("cannot convert %v (type %s) to type %s", t1.Constant, t1, t2)
			}
			return nil, err
		}
		t1.setValue(t2.Type)
		t1 = &TypeInfo{Type: t2.Type, Constant: c}
	} else if t2.Untyped() {
		c, err := tc.convertImplicitly(t2, t1.Type)
		if err != nil {
			if err == errNotRepresentable {
				err = fmt.Errorf("cannot convert %v (type %s) to type %s", t2.Constant, t2, t1)
			}
			return nil, err
		}
		t2.setValue(t1.Type)
		t2 = &TypeInfo{Type: t1.Type, Constant: c}
	}

	if t1.IsConstant() && !t2.IsConstant() {
		t1.setValue(nil)
	}

	if !t1.IsConstant() && t2.IsConstant() {
		t2.setValue(nil)
	}

	if t1.IsConstant() && t2.IsConstant() {
		if t1.Type != t2.Type {
			return nil, fmt.Errorf("mismatched types %s and %s", t1, t2)
		}
		c, err := t1.Constant.binaryOp(op, t2.Constant)
		if err != nil {
			if err == errInvalidOperation {
				switch op {
				case ast.OperatorModulo:
					if isComplex(t1.Type.Kind()) {
						return nil, errors.New("complex % operation")
					}
					return nil, errors.New("floating-point % operation")
				case ast.OperatorAnd, ast.OperatorOr, ast.OperatorXor:
					if isComplex(t1.Type.Kind()) {
						return nil, fmt.Errorf("operator %s not defined on untyped complex", op)
					}
					return nil, fmt.Errorf("operator %s not defined on untyped float", op)
				}
				return nil, fmt.Errorf("operator %s not defined on %s", op, t1.ShortString())
			}
			return nil, err
		}
		if evalToBoolOperators[op] {
			ti := &TypeInfo{Type: boolType, Constant: c}
			if isComparison(op) {
				ti.Properties = PropertyUntyped
			}
			return ti, nil
		}
		ti := &TypeInfo{Type: t1.Type, Constant: c}
		ti.Constant, err = tc.convertExplicitly(ti, t1.Type)
		if err != nil {
			return nil, err
		}
		return ti, nil
	}

	if isComparison(op) {
		if tc.isAssignableTo(t1, expr1, t2.Type) != nil && tc.isAssignableTo(t2, expr2, t1.Type) != nil {
			return nil, fmt.Errorf("mismatched types %s and %s", t1.ShortString(), t2.ShortString())
		}
		if op == ast.OperatorEqual || op == ast.OperatorNotEqual {
			if !t1.Type.Comparable() {
				// https://github.com/open2b/scriggo/issues/368
				return nil, fmt.Errorf("%s cannot be compared", t1.Type)
			}
			if !t2.Type.Comparable() {
				// https://github.com/open2b/scriggo/issues/368
				return nil, fmt.Errorf("%s cannot be compared", t2.Type)
			}
		} else if !isOrdered(t1) {
			return nil, fmt.Errorf("operator %s not defined on %s", op, t1.Type.Kind())
		}
		return &TypeInfo{Type: boolType, Properties: PropertyUntyped}, nil
	}

	if t1.Type != t2.Type {
		return nil, fmt.Errorf("mismatched types %s and %s", t1.ShortString(), t2.ShortString())
	}

	if kind := t1.Type.Kind(); !operatorsOfKind[kind][op] {
		return nil, fmt.Errorf("operator %s not defined on %s)", op, kind)
	}

	return &TypeInfo{Type: t1.Type}, nil
}

// checkSize checks the type of expr as a make size parameter.
// If it is a constant returns the integer value, otherwise returns -1.
func (tc *typechecker) checkSize(expr ast.Expression, typ reflect.Type, name string) constant {
	size := tc.checkExpr(expr)
	if size.Untyped() && !size.IsNumeric() || !size.Untyped() && !size.IsInteger() {
		got := size.String()
		if name == "size" || name == "buffer" {
			if size.Nil() {
				panic(tc.errorf(expr, "cannot convert nil to type int"))
			}
			got = size.ShortString()
		}
		panic(tc.errorf(expr, "non-integer %s argument in make(%s) - %s", name, typ, got))
	}
	size.setValue(intType)
	if size.IsConstant() {
		c, err := tc.convertExplicitly(size, intType)
		if err != nil {
			panic(tc.errorf(expr, "%s", err))
		}
		if c.int64() < 0 {
			panic(tc.errorf(expr, "negative %s argument in make(%s)", name, typ))
		}
		return c
	}
	return nil
}

// checkBuiltinCall checks the builtin call expr, returning the list of results.
func (tc *typechecker) checkBuiltinCall(expr *ast.Call) []*TypeInfo {

	ident := expr.Func.(*ast.Identifier)

	if expr.IsVariadic && ident.Name != "append" {
		panic(tc.errorf(expr, "invalid use of ... with builtin %s", ident.Name))
	}

	switch ident.Name {

	case "append":
		if len(expr.Args) == 0 {
			panic(tc.errorf(expr, "missing arguments to append"))
		}
		slice := tc.checkExpr(expr.Args[0])
		if slice.Nil() {
			panic(tc.errorf(expr, "first argument to append must be typed slice; have untyped nil"))
		}
		if slice.Type.Kind() != reflect.Slice {
			panic(tc.errorf(expr, "first argument to append must be slice; have %s", slice.StringWithNumber(true)))
		}
		if expr.IsVariadic {
			if len(expr.Args) == 1 {
				panic(tc.errorf(expr, "cannot use ... on first argument to append"))
			} else if len(expr.Args) > 2 {
				panic(tc.errorf(expr, "too many arguments to append"))
			}
			t := tc.checkExpr(expr.Args[1])
			isSpecialCase := t.Type.Kind() == reflect.String && slice.Type.Elem() == uint8Type
			if !isSpecialCase && tc.isAssignableTo(t, expr.Args[1], slice.Type) != nil {
				panic(tc.errorf(expr, "cannot use %s (type %s) as type %s in append", expr.Args[1], t, slice.Type))
			}
		} else if len(expr.Args) > 1 {
			elemType := slice.Type.Elem()
			for i, el := range expr.Args {
				if i == 0 {
					continue
				}
				t := tc.checkExpr(el)
				if err := tc.isAssignableTo(t, el, elemType); err != nil {
					if _, ok := err.(invalidTypeInAssignment); ok {
						panic(tc.errorf(expr, "%s in append", err))
					}
					panic(tc.errorf(expr, "%s", err))
				}
				t.setValue(elemType)
			}
		}
		return []*TypeInfo{{Type: slice.Type}}

	case "cap":
		if len(expr.Args) < 1 {
			panic(tc.errorf(expr, "missing argument to cap: %s", expr))
		}
		if len(expr.Args) > 1 {
			panic(tc.errorf(expr, "too many arguments to cap: %s", expr))
		}
		arg := expr.Args[0]
		t := tc.checkExpr(arg)
		if t.Nil() {
			panic(tc.errorf(expr, "use of untyped nil"))
		}
		ti := &TypeInfo{Type: intType}
		switch typ := t.Type; typ.Kind() {
		case reflect.Array:
			if tc.isCompileConstant(arg) {
				ti.Constant = newIntConst(int64(typ.Len()))
			}
		case reflect.Chan, reflect.Slice:
		case reflect.Ptr:
			if typ.Elem().Kind() != reflect.Array {
				panic(tc.errorf(expr, "invalid argument %s (type %s) for cap", arg, t.ShortString()))
			}
			if tc.isCompileConstant(arg) {
				ti.Constant = newIntConst(int64(typ.Elem().Len()))
			}
		default:
			panic(tc.errorf(expr, "invalid argument %s (type %s) for cap", arg, t.ShortString()))
		}
		return []*TypeInfo{ti}

	case "close":
		if len(expr.Args) == 0 {
			panic(tc.errorf(expr, "missing argument to close: %s", expr))
		}
		if len(expr.Args) > 1 {
			panic(tc.errorf(expr, "too many arguments to close: %s", expr))
		}
		arg := tc.checkExpr(expr.Args[0])
		if arg.Nil() {
			panic(tc.errorf(expr, "use of untyped nil"))
		}
		if arg.Type.Kind() != reflect.Chan {
			panic(tc.errorf(expr, "invalid operation: %s (non-chan type %s)", expr, arg))
		}
		if arg.Type.ChanDir() == reflect.RecvDir {
			panic(tc.errorf(expr, "invalid operation: %s (cannot close receive-only channel)", expr))
		}
		return []*TypeInfo{}

	case "complex":
		switch len(expr.Args) {
		case 0:
			panic(tc.errorf(expr, "missing argument to complex - complex(<N>, <N>)"))
		case 1:
			panic(tc.errorf(expr, "invalid operation: complex expects two arguments"))
		case 2:
		default:
			panic(tc.errorf(expr, "too many arguments to complex - complex(%s, <N>)", expr.Args[0]))
		}
		re := tc.checkExpr(expr.Args[0])
		im := tc.checkExpr(expr.Args[1])
		reKind := re.Type.Kind()
		imKind := im.Type.Kind()
		if re.IsUntypedConstant() && im.IsUntypedConstant() {
			if !isNumeric(reKind) || !isNumeric(imKind) {
				if reKind == imKind {
					panic(tc.errorf(expr, "invalid operation: %s (arguments have type %s, expected floating-point)", expr, re))
				}
				panic(tc.errorf(expr, "invalid operation: %s (mismatched types %s and %s)", expr, re, im))
			}
			if !re.Constant.imag().zero() {
				panic(tc.errorf(expr, "constant %s truncated to real", expr.Args[0]))
			}
			if !im.Constant.imag().zero() {
				panic(tc.errorf(expr, "constant %s truncated to real", expr.Args[1]))
			}
			ti := &TypeInfo{
				Type:       complex128Type,
				Constant:   newComplexConst(re.Constant, im.Constant),
				Properties: PropertyUntyped,
			}
			return []*TypeInfo{ti}
		}
		if re.IsUntypedConstant() {
			k := im.Type.Kind()
			_ = k
			c, err := tc.convertImplicitly(re, im.Type)
			if err != nil {
				panic(tc.errorf(expr, "cannot convert %s (type %s) to type %s", re.Constant, re, im))
			}
			re = &TypeInfo{Type: im.Type, Constant: c}
		} else if im.IsUntypedConstant() {
			c, err := tc.convertImplicitly(im, re.Type)
			if err != nil {
				panic(tc.errorf(expr, "cannot convert %s (type %s) to type %s", im.Constant, im, re))
			}
			im = &TypeInfo{Type: re.Type, Constant: c}
		} else if reKind != imKind {
			panic(tc.errorf(expr, "invalid operation: %s (mismatched types %s and %s)", expr, re, im))
		}
		ti := &TypeInfo{}
		switch re.Type.Kind() {
		case reflect.Float32:
			ti.Type = complex64Type
		case reflect.Float64:
			ti.Type = complex128Type
		default:
			panic(tc.errorf(expr, "invalid operation: %s (arguments have type %s, expected floating-point)", expr, re.Type))
		}
		if re.IsConstant() && im.IsConstant() {
			ti.Constant = newComplexConst(re.Constant, im.Constant)
		}
		return []*TypeInfo{ti}

	case "copy":
		if len(expr.Args) < 2 {
			panic(tc.errorf(expr, "missing argument to copy: %s", expr))
		}
		if len(expr.Args) > 2 {
			panic(tc.errorf(expr, "too many arguments to copy: %s", expr))
		}
		dst := tc.checkExpr(expr.Args[0])
		src := tc.checkExpr(expr.Args[1])
		if dst.Nil() || src.Nil() {
			panic(tc.errorf(expr, "use of untyped nil"))
		}
		dk := dst.Type.Kind()
		sk := src.Type.Kind()
		if dk != reflect.Slice && sk != reflect.Slice {
			panic(tc.errorf(expr, "arguments to copy must be slices; have %s, %s", dst.ShortString(), src.ShortString()))
		}
		if dk != reflect.Slice {
			panic(tc.errorf(expr, "first argument to copy should be slice; have %s", dst.ShortString()))
		}
		if sk != reflect.Slice && sk != reflect.String {
			panic(tc.errorf(expr, "second argument to copy should be slice or string; have %s", src.ShortString()))
		}
		if (sk == reflect.String && dst.Type.Elem() != uint8Type) || (sk == reflect.Slice && dst.Type.Elem() != src.Type.Elem()) {
			panic(tc.errorf(expr, "arguments to copy have different element types: %s and %s", dst, src))
		}
		return []*TypeInfo{{Type: intType}}

	case "delete":
		switch len(expr.Args) {
		case 0:
			panic(tc.errorf(expr, "missing arguments to delete"))
		case 1:
			panic(tc.errorf(expr, "missing second (key) argument to delete"))
		case 2:
		default:
			panic(tc.errorf(expr, "too many arguments to delete"))
		}
		t := tc.checkExpr(expr.Args[0])
		key := tc.checkExpr(expr.Args[1])
		if t.Nil() {
			panic(tc.errorf(expr, "first argument to delete must be map; have nil"))
		}
		if t.Type.Kind() != reflect.Map {
			panic(tc.errorf(expr, "first argument to delete must be map; have %s", t))
		}
		keyType := t.Type.Key()
		if err := tc.isAssignableTo(key, expr.Args[1], keyType); err != nil {
			if _, ok := err.(invalidTypeInAssignment); ok {
				panic(tc.errorf(expr, "%s in delete", err))
			}
			if _, ok := err.(nilConvertionError); ok {
				panic(tc.errorf(expr, "cannot use nil as type %s in delete", keyType))
			}
			panic(tc.errorf(expr, "%s", err))
		}
		if key.IsConstant() {
			_, err := tc.convertExplicitly(key, keyType)
			if err != nil {
				panic(tc.errorf(expr, "%s", err))
			}
		}
		key.setValue(keyType)
		return nil

	case "len":
		if len(expr.Args) < 1 {
			panic(tc.errorf(expr, "missing argument to len: %s", expr))
		}
		if len(expr.Args) > 1 {
			panic(tc.errorf(expr, "too many arguments to len: %s", expr))
		}
		arg := expr.Args[0]
		t := tc.checkExpr(arg)
		if t.Nil() {
			panic(tc.errorf(expr, "use of untyped nil"))
		}
		t.setValue(nil)
		ti := &TypeInfo{Type: intType}
		ti.setValue(nil)
		switch typ := t.Type; typ.Kind() {
		case reflect.Array:
			if tc.isCompileConstant(arg) {
				ti.Constant = newIntConst(int64(typ.Len()))
			}
		case reflect.Chan, reflect.Map, reflect.Slice:
		case reflect.String:
			if t.IsConstant() {
				ti.Constant = newIntConst(int64(len(t.Constant.String())))
			}
		case reflect.Ptr:
			if typ.Elem().Kind() != reflect.Array {
				panic(tc.errorf(expr, "invalid argument %s (type %s) for len", arg, t.ShortString()))
			}
			if tc.isCompileConstant(arg) {
				ti.Constant = newIntConst(int64(typ.Elem().Len()))
			}
		default:
			panic(tc.errorf(expr, "invalid argument %s (type %s) for len", arg, t.ShortString()))
		}
		return []*TypeInfo{ti}

	case "make":
		numArgs := len(expr.Args)
		if numArgs == 0 {
			panic(tc.errorf(expr, "missing argument to make"))
		}
		t := tc.checkType(expr.Args[0])
		switch t.Type.Kind() {
		case reflect.Slice:
			if numArgs == 1 {
				panic(tc.errorf(expr, "missing len argument to make(%s)", expr.Args[0]))
			}
			if numArgs > 1 {
				l := tc.checkSize(expr.Args[1], t.Type, "len")
				if numArgs > 2 {
					c := tc.checkSize(expr.Args[2], t.Type, "cap")
					if c != nil {
						if l != nil && l.int64() > c.int64() {
							panic(tc.errorf(expr, "len larger than cap in make(%s)", t.Type))
						}
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
				tc.checkSize(expr.Args[1], t.Type, "size")
			}
		case reflect.Chan:
			if numArgs > 2 {
				panic(tc.errorf(expr, "too many arguments to make(%s)", expr.Args[0]))
			}
			if numArgs == 2 {
				tc.checkSize(expr.Args[1], t.Type, "buffer")
			}
		default:
			panic(tc.errorf(expr, "cannot make type %s", t))
		}
		return []*TypeInfo{{Type: t.Type}}

	case "new":
		if len(expr.Args) == 0 {
			panic(tc.errorf(expr, "missing argument to new"))
		}
		t := tc.checkType(expr.Args[0])
		if len(expr.Args) > 1 {
			panic(tc.errorf(expr, "too many arguments to new(%s)", expr.Args[0]))
		}
		return []*TypeInfo{{Type: tc.types.PtrTo(t.Type)}}

	case "panic":
		if len(expr.Args) == 0 {
			panic(tc.errorf(expr, "missing argument to panic: panic()"))
		}
		if len(expr.Args) > 1 {
			panic(tc.errorf(expr, "too many arguments to panic: %s", expr))
		}
		ti := tc.checkExpr(expr.Args[0])
		if ti.Nil() {
			ti = tc.nilOf(emptyInterfaceType)
			tc.typeInfos[expr.Args[0]] = ti
		} else {
			ti.setValue(nil)
		}
		return nil

	case "print", "println":
		for _, arg := range expr.Args {
			tc.checkExpr(arg)
			tc.typeInfos[arg].setValue(nil)
		}
		return nil

	case "real", "imag":
		switch len(expr.Args) {
		case 0:
			panic(tc.errorf(expr, "missing argument to %s: %s()", ident.Name, ident.Name))
		case 1:
		default:
			panic(tc.errorf(expr, "too many arguments to %s: %s", ident.Name, expr))
		}
		t := tc.checkExpr(expr.Args[0])
		ti := &TypeInfo{Type: float64Type}
		if t.IsUntypedConstant() {
			if !isNumeric(t.Type.Kind()) {
				panic(tc.errorf(expr, "invalid argument %s (type %s) for %s", expr.Args[0], t, ident.Name))
			}
			ti.Properties = PropertyUntyped
		} else {
			switch t.Type.Kind() {
			case reflect.Complex64:
				ti.Type = float32Type
			case reflect.Complex128:
			default:
				panic(tc.errorf(expr, "invalid argument %s (type %s) for %s", expr.Args[0], t.Type, ident.Name))
			}
		}
		if t.IsConstant() {
			if ident.Name == "real" {
				ti.Constant = t.Constant.real()
			} else {
				ti.Constant = t.Constant.imag()
			}
		}
		return []*TypeInfo{ti}

	case "recover":
		if len(expr.Args) > 0 {
			panic(tc.errorf(expr, "too many arguments to recover"))
		}
		return []*TypeInfo{{Type: emptyInterfaceType}}

	}

	panic(fmt.Sprintf("unexpected builtin %s", ident.Name))

}

// checkCallExpression type checks a call expression, including type
// conversions and built-in function calls. Returns a list of typeinfos
// obtained from the call and returns two booleans indicating respectively if
// expr is a builtin call or a conversion.
func (tc *typechecker) checkCallExpression(expr *ast.Call, statement bool) ([]*TypeInfo, bool, bool) {

	// Check a builtin function call.
	if ident, ok := expr.Func.(*ast.Identifier); ok {
		if ti, ok := tc.lookupScopes(ident.Name, false); ok && ti.IsBuiltinFunction() {
			tc.typeInfos[expr.Func] = ti
			return tc.checkBuiltinCall(expr), true, false
		}
	}

	t := tc.checkExprOrType(expr.Func)

	switch t.MethodType {
	case MethodValueConcrete:
		t.MethodType = MethodCallConcrete
	case MethodValueInterface:
		t.MethodType = MethodCallInterface
	}

	// expr is a ShowMacro expression which is not defined and which has been
	// marked as "to be ignored" or "to do".
	if t == showMacroIgnoredTi {
		return nil, false, false
	}

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
		arg := tc.checkExpr(expr.Args[0])
		c, err := tc.convertExplicitly(arg, t.Type)
		if err != nil {
			if err == errTypeConversion {
				panic(tc.errorf(expr, "cannot convert %s (type %s) to type %s", expr.Args[0], arg.Type, t.Type))
			}
			panic(tc.errorf(expr, "%s", err))
		}
		if arg.Nil() {
			return []*TypeInfo{tc.nilOf(t.Type)}, false, true
		}
		converted := &TypeInfo{Type: t.Type, Constant: c}
		if c == nil {
			arg.setValue(arg.Type)
		} else {
			converted.setValue(t.Type)
		}
		return []*TypeInfo{converted}, false, true
	}

	if t.Type.Kind() != reflect.Func {
		panic(tc.errorf(expr, "cannot call non-function %v (type %s)", expr.Func, t))
	}

	var funcIsVariadic = t.Type.IsVariadic()
	var callIsVariadic = expr.IsVariadic

	if !funcIsVariadic && callIsVariadic {
		panic(tc.errorf(expr, "invalid use of ... in call to %s", expr.Func))
	}

	args := expr.Args
	numIn := t.Type.NumIn()

	isSpecialCase := false

	if len(args) == 1 && numIn > 1 && !callIsVariadic {
		if c, ok := args[0].(*ast.Call); ok {
			isSpecialCase = true
			args = nil
			tis, _, _ := tc.checkCallExpression(c, false)
			if len(tis) == 1 {
				tc.typeInfos[c] = tis[0]
			}
			for _, ti := range tis {
				v := ast.NewCall(c.Pos(), c.Func, c.Args, false)
				tc.typeInfos[v] = ti
				args = append(args, v)
			}
		}
	} else if len(args) == 1 && numIn == 1 && funcIsVariadic && !callIsVariadic {
		if c, ok := args[0].(*ast.Call); ok {
			args = nil
			tis, _, _ := tc.checkCallExpression(c, false)
			if len(tis) == 1 {
				tc.typeInfos[c] = tis[0]
			}
			for _, ti := range tis {
				v := ast.NewCall(c.Pos(), c.Func, c.Args, false)
				tc.typeInfos[v] = ti
				args = append(args, v)
			}
		}
	}

	if (!funcIsVariadic && len(args) != numIn) || (funcIsVariadic && len(args) < numIn-1) {
		have := "("
		for i, arg := range args {
			if i > 0 {
				have += ", "
			}
			c := tc.typeInfos[arg]
			if c == nil {
				c = tc.checkExpr(arg)
			}
			if c.Nil() {
				have += "nil"
			} else {
				have += c.StringWithNumber(false)
			}
		}
		have += ")"
		want := "("
		for i := 0; i < numIn; i++ {
			if i > 0 {
				want += ", "
			}
			in := t.Type.In(i)
			if i == numIn-1 && funcIsVariadic {
				want += "..."
				in = in.Elem()
			}
			want += in.String()
		}
		want += ")"
		if len(args) < numIn {
			panic(tc.errorf(expr, "not enough arguments in call to %s\n\thave %s\n\twant %s", expr.Func, have, want))
		}
		panic(tc.errorf(expr, "too many arguments in call to %s\n\thave %s\n\twant %s", expr.Func, have, want))
	}

	var in reflect.Type
	var lastIn = numIn - 1

	for i, arg := range args {
		if i < lastIn || !funcIsVariadic {
			in = t.Type.In(i)
		} else if i == lastIn {
			in = t.Type.In(lastIn).Elem()
		}
		if isSpecialCase {
			a := tc.typeInfos[arg]
			if err := tc.isAssignableTo(a, arg, in); err != nil {
				if _, ok := err.(invalidTypeInAssignment); ok {
					panic(tc.errorf(args[i], "cannot use %s as type %s in argument to %s", a, in, expr.Func))
				}
				panic(tc.errorf(expr, "%s", err))
			}
			continue
		}
		a := tc.checkExpr(arg)
		if i == lastIn && callIsVariadic {
			if err := tc.isAssignableTo(a, arg, tc.types.SliceOf(in)); err != nil {
				if _, ok := err.(invalidTypeInAssignment); ok {
					panic(tc.errorf(expr, "%s in argument to %s", err, expr.Func))
				}
				panic(tc.errorf(expr, "%s", err))
			}
			continue
		}
		if err := tc.isAssignableTo(a, arg, in); err != nil {
			if _, ok := err.(invalidTypeInAssignment); ok {
				panic(tc.errorf(expr, "%s in argument to %s", err, expr.Func))
			}
			if _, ok := err.(nilConvertionError); ok {
				panic(tc.errorf(args[i], "cannot use %s as type %s in argument to %s", a, in, expr.Func))
			}
			panic(tc.errorf(expr, "%s", err))
		}
		if a.Nil() {
			a := tc.nilOf(in)
			tc.typeInfos[expr.Args[i]] = a
		} else {
			a.setValue(in)
		}
	}

	numOut := t.Type.NumOut()
	resultTypes := make([]*TypeInfo, numOut)
	for i := 0; i < numOut; i++ {
		resultTypes[i] = &TypeInfo{Type: t.Type.Out(i)}
	}

	return resultTypes, false, false
}

// maxIndex returns the maximum element index in the composite literal node.
func (tc *typechecker) maxIndex(node *ast.CompositeLiteral) int {
	if len(node.KeyValues) == 0 {
		return -1
	}
	maxIndex := 0
	currentIndex := -1
	for _, kv := range node.KeyValues {
		if kv.Key == nil {
			currentIndex++
		} else {
			currentIndex = -1
			ti := tc.checkExpr(kv.Key)
			if ti.IsConstant() {
				c, _ := ti.Constant.representedBy(intType)
				if c != nil {
					currentIndex = int(c.int64())
				}
			}
			if currentIndex < 0 {
				panic(tc.errorf(kv.Key, "index must be non-negative integer constant"))
			}
		}
		if currentIndex > maxIndex {
			maxIndex = currentIndex
		}
	}
	return maxIndex
}

// checkCompositeLiteral type checks a composite literal. typ is the type of
// the composite literal.
func (tc *typechecker) checkCompositeLiteral(node *ast.CompositeLiteral, typ reflect.Type) *TypeInfo {

	// Handle composite literal nodes with implicit type.
	if node.Type == nil {
		tc.typeInfos[node.Type] = &TypeInfo{Properties: PropertyIsType, Type: typ}
	}

	maxIndex := -1
	switch node.Type.(type) {
	case *ast.ArrayType, *ast.SliceType:
		maxIndex = tc.maxIndex(node)
	}

	// Check the type of the composite literal. Arrays are handled in a special
	// way since composite literals support array types declarations with
	// ellipses as length.
	var ti *TypeInfo
	if array, ok := node.Type.(*ast.ArrayType); ok {
		ti = tc.checkArrayType(array, maxIndex+1)
	} else {
		ti = tc.checkType(node.Type)
	}
	// tc.typeInfos[node.Type] = ti

	switch ti.Type.Kind() {

	case reflect.Struct:

		declType := 0
		for _, kv := range node.KeyValues {
			if kv.Key == nil {
				if declType == 1 {
					panic(tc.errorf(node, "mixture of field:value and value initializers"))
				}
				declType = -1
				continue
			} else {
				if declType == -1 {
					panic(tc.errorf(node, "mixture of field:value and value initializers"))
				}
				declType = 1
			}
		}
		switch declType == 1 {
		case true: // struct with explicit fields.
			hasField := map[string]struct{}{}
			for i := range node.KeyValues {
				keyValue := &node.KeyValues[i]
				ident, ok := keyValue.Key.(*ast.Identifier)
				if !ok || ident.Name == "_" {
					panic(tc.errorf(node, "invalid field name %s in struct initializer", keyValue.Key))
				}
				if _, ok := hasField[ident.Name]; ok {
					panic(tc.errorf(node, "duplicate field name in struct literal: %s", keyValue.Key))
				}
				hasField[ident.Name] = struct{}{}
				fieldTi, newName, ok := tc.fieldByName(ti, ident.Name)
				if !ok {
					panic(tc.errorf(node, "unknown field '%s' in struct literal of type %s", keyValue.Key, ti))
				}
				valueTi := tc.checkExpr(keyValue.Value)
				if err := tc.isAssignableTo(valueTi, keyValue.Value, fieldTi.Type); err != nil {
					if _, ok := err.(invalidTypeInAssignment); ok {
						panic(tc.errorf(node, "%s in field value", err))
					}
					panic(tc.errorf(node, "%s", err))
				}
				if valueTi.Nil() {
					valueTi = tc.nilOf(fieldTi.Type)
					tc.typeInfos[keyValue.Value] = valueTi
				} else {
					valueTi.setValue(fieldTi.Type)
				}
				ident.Name = newName
			}
		case false: // struct with implicit fields.
			if len(node.KeyValues) == 0 {
				ti := &TypeInfo{Type: ti.Type}
				tc.typeInfos[node] = ti
				return ti
			}
			if len(node.KeyValues) < ti.Type.NumField() {
				panic(tc.errorf(node, "too few values in %s literal", ti))
			}
			if len(node.KeyValues) > ti.Type.NumField() {
				panic(tc.errorf(node, "too many values in %s literal", ti))
			}
			for i := range node.KeyValues {
				keyValue := &node.KeyValues[i]
				valueTi := tc.checkExpr(keyValue.Value)
				fieldTi := ti.Type.Field(i)
				if err := tc.isAssignableTo(valueTi, keyValue.Value, fieldTi.Type); err != nil {
					if _, ok := err.(invalidTypeInAssignment); ok {
						panic(tc.errorf(node, "%s in field value", err))
					}
					panic(tc.errorf(node, "%s", err))
				}
				if !isExported(fieldTi.Name) {
					panic(tc.errorf(node, "implicit assignment of unexported field '%s' in %v", fieldTi.Name, node))
				}
				keyValue.Key = ast.NewIdentifier(node.Pos(), fieldTi.Name)
				if valueTi.Nil() {
					valueTi = tc.nilOf(fieldTi.Type)
					tc.typeInfos[keyValue.Value] = valueTi
				} else {
					valueTi.setValue(fieldTi.Type)
				}
			}
		}

	case reflect.Array:

		hasIndex := map[int]struct{}{}
		for i := range node.KeyValues {
			kv := &node.KeyValues[i]
			if kv.Key != nil {
				keyTi := tc.checkExpr(kv.Key)
				if keyTi.Constant == nil {
					panic(tc.errorf(node, "index must be non-negative integer constant"))
				}
				if keyTi.IsConstant() {
					index := int(keyTi.Constant.int64())
					if _, ok := hasIndex[index]; ok {
						panic(tc.errorf(node, "duplicate index in array literal: %s", kv.Key))
					}
					hasIndex[index] = struct{}{}
				}
			}
			var elemTi *TypeInfo
			if cl, ok := kv.Value.(*ast.CompositeLiteral); ok {
				elemTi = tc.checkCompositeLiteral(cl, ti.Type.Elem())
			} else {
				elemTi = tc.checkExpr(kv.Value)
			}
			if err := tc.isAssignableTo(elemTi, kv.Value, ti.Type.Elem()); err != nil {
				k := ti.Type.Elem().Kind()
				if _, ok := err.(invalidTypeInAssignment); ok && k == reflect.Slice || k == reflect.Array {
					panic(tc.errorf(node, "%s in array or slice literal", err))
				}
				panic(tc.errorf(node, "%s", err))
			}
			if elemTi.Nil() {
				elemTi = tc.nilOf(ti.Type.Elem())
				tc.typeInfos[kv.Value] = elemTi
			} else {
				elemTi.setValue(ti.Type.Elem())
			}
		}

	case reflect.Slice:

		hasIndex := map[int]struct{}{}
		for i := range node.KeyValues {
			kv := &node.KeyValues[i]
			if kv.Key != nil {
				keyTi := tc.checkExpr(kv.Key)
				if keyTi.Constant == nil {
					panic(tc.errorf(node, "index must be non-negative integer constant"))
				}
				if keyTi.IsConstant() {
					index := int(keyTi.Constant.int64())
					if _, ok := hasIndex[index]; ok {
						panic(tc.errorf(node, "duplicate index in array literal: %s", kv.Key))
					}
					hasIndex[index] = struct{}{}
				}
			}
			var elemTi *TypeInfo
			if cl, ok := kv.Value.(*ast.CompositeLiteral); ok {
				elemTi = tc.checkCompositeLiteral(cl, ti.Type.Elem())
			} else {
				elemTi = tc.checkExpr(kv.Value)
			}
			if err := tc.isAssignableTo(elemTi, kv.Value, ti.Type.Elem()); err != nil {
				k := ti.Type.Elem().Kind()
				if _, ok := err.(invalidTypeInAssignment); ok {
					if k == reflect.Slice || k == reflect.Array {
						panic(tc.errorf(node, "%s in array or slice literal", err))
					}
					panic(tc.errorf(node, "cannot convert %s (type %s) to type %v", kv.Value, elemTi, ti.Type.Elem()))
				}
				panic(tc.errorf(node, "%s", err))
			}
			if elemTi.Nil() {
				elemTi = tc.nilOf(ti.Type.Elem())
				tc.typeInfos[kv.Value] = elemTi
			} else {
				elemTi.setValue(ti.Type.Elem())
			}
		}

	case reflect.Map:

		hasKey := map[interface{}]struct{}{}
		for i := range node.KeyValues {
			kv := &node.KeyValues[i]
			keyType := ti.Type.Key()
			elemType := ti.Type.Elem()
			if kv.Key == nil {
				panic(tc.errorf(node, "missing key in map literal"))
			}
			var keyTi *TypeInfo
			if compLit, ok := kv.Key.(*ast.CompositeLiteral); ok {
				keyTi = tc.checkCompositeLiteral(compLit, keyType)
			} else {
				keyTi = tc.checkExpr(kv.Key)
			}
			if err := tc.isAssignableTo(keyTi, kv.Key, keyType); err != nil {
				if _, ok := err.(invalidTypeInAssignment); ok {
					panic(tc.errorf(node, "%s in map key", err))
				}
				panic(tc.errorf(node, "%s", err))
			}
			if keyTi.IsConstant() {
				key := tc.typedValue(keyTi, keyType)
				if _, ok := hasKey[key]; ok {
					panic(tc.errorf(node, "duplicate key %s in map literal", kv.Key))
				}
				hasKey[key] = struct{}{}
			}
			if keyTi.Nil() {
				keyTi = tc.nilOf(keyType)
				tc.typeInfos[kv.Key] = keyTi
			} else {
				keyTi.setValue(keyType)
			}
			var valueTi *TypeInfo
			if cl, ok := kv.Value.(*ast.CompositeLiteral); ok {
				valueTi = tc.checkCompositeLiteral(cl, elemType)
			} else {
				valueTi = tc.checkExpr(kv.Value)
			}
			if err := tc.isAssignableTo(valueTi, kv.Value, elemType); err != nil {
				if _, ok := err.(invalidTypeInAssignment); ok {
					panic(tc.errorf(node, "%s in map value", err))
				}
				panic(tc.errorf(node, "%s", err))
			}
			if valueTi.Nil() {
				valueTi = tc.nilOf(elemType)
				tc.typeInfos[kv.Value] = valueTi
			} else {
				valueTi.setValue(elemType)
			}
		}

	}

	nodeTi := &TypeInfo{Type: ti.Type}
	tc.typeInfos[node] = nodeTi

	return nodeTi
}

// builtinCallName returns the name of the builtin function in a call
// expression. If expr is not a call expression or the function is not a
// builtin, it returns an empty string.
func (tc *typechecker) builtinCallName(expr ast.Expression) string {
	if call, ok := expr.(*ast.Call); ok && tc.typeInfos[call.Func].IsBuiltinFunction() {
		return call.Func.(*ast.Identifier).Name
	}
	return ""
}

// isCompileConstant reports whether expr does not contain channel receives or
// non-constant function calls.
func (tc *typechecker) isCompileConstant(expr ast.Expression) bool {
	if ti := tc.typeInfos[expr]; ti.IsConstant() {
		return true
	}
	switch expr := expr.(type) {
	case *ast.BinaryOperator:
		return tc.isCompileConstant(expr.Expr1) && tc.isCompileConstant(expr.Expr2)
	case *ast.Call:
		ti := tc.typeInfos[expr.Func]
		if ti.IsType() {
			return tc.isCompileConstant(expr.Args[0])
		}
		switch tc.builtinCallName(expr) {
		case "len", "cap":
			return tc.isCompileConstant(expr.Args[0])
		case "make":
			switch len(expr.Args) {
			case 1:
				return true
			case 2:
				return tc.isCompileConstant(expr.Args[0])
			case 3:
				return tc.isCompileConstant(expr.Args[0]) && tc.isCompileConstant(expr.Args[1])
			}
		}
		return false
	case *ast.CompositeLiteral:
		for _, kv := range expr.KeyValues {
			if kv.Key != nil && !tc.isCompileConstant(kv.Key) {
				return false
			}
			if !tc.isCompileConstant(kv.Value) {
				return false
			}
		}
	case *ast.Index:
		return tc.isCompileConstant(expr.Expr) && tc.isCompileConstant(expr.Index)
	case *ast.Selector:
		return tc.isCompileConstant(expr.Expr)
	case *ast.Slicing:
		return tc.isCompileConstant(expr.Expr) && tc.isCompileConstant(expr.Low) &&
			tc.isCompileConstant(expr.High) && (!expr.IsFull || tc.isCompileConstant(expr.Max))
	case *ast.TypeAssertion:
		return tc.isCompileConstant(expr.Expr)
	case *ast.UnaryOperator:
		return expr.Op != ast.OperatorReceive && tc.isCompileConstant(expr.Expr)
	}
	return true
}
