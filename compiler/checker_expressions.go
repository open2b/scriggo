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
	"strings"
	"unicode"
	"unicode/utf8"

	"scriggo/compiler/ast"
	"scriggo/runtime"
)

type scopeElement struct {
	t    *typeInfo
	decl *ast.Identifier
}

type typeCheckerScope map[string]scopeElement

var boolType = reflect.TypeOf(false)
var uintType = reflect.TypeOf(uint(0))
var uint8Type = reflect.TypeOf(uint8(0))
var int32Type = reflect.TypeOf(int32(0))
var byteSliceType = reflect.TypeOf([]byte(nil))

var uint8TypeInfo = &typeInfo{Type: uint8Type, Properties: propertyIsType | propertyPredeclared}
var int32TypeInfo = &typeInfo{Type: int32Type, Properties: propertyIsType | propertyPredeclared}

var untypedBoolTypeInfo = &typeInfo{Type: boolType, Properties: propertyUntyped}

var envType = reflect.TypeOf(&runtime.Env{})

var universe = typeCheckerScope{
	"$notZero":   {t: &typeInfo{Properties: propertyPredeclared}},
	"append":     {t: &typeInfo{Properties: propertyPredeclared}},
	"cap":        {t: &typeInfo{Properties: propertyPredeclared}},
	"close":      {t: &typeInfo{Properties: propertyPredeclared}},
	"complex":    {t: &typeInfo{Properties: propertyPredeclared}},
	"copy":       {t: &typeInfo{Properties: propertyPredeclared}},
	"delete":     {t: &typeInfo{Properties: propertyPredeclared}},
	"imag":       {t: &typeInfo{Properties: propertyPredeclared}},
	"iota":       {t: &typeInfo{Properties: propertyPredeclared, Type: intType}},
	"len":        {t: &typeInfo{Properties: propertyPredeclared}},
	"make":       {t: &typeInfo{Properties: propertyPredeclared}},
	"new":        {t: &typeInfo{Properties: propertyPredeclared}},
	"nil":        {t: &typeInfo{Properties: propertyUntyped | propertyPredeclared}},
	"panic":      {t: &typeInfo{Properties: propertyPredeclared}},
	"print":      {t: &typeInfo{Properties: propertyPredeclared}},
	"println":    {t: &typeInfo{Properties: propertyPredeclared}},
	"real":       {t: &typeInfo{Properties: propertyPredeclared}},
	"recover":    {t: &typeInfo{Properties: propertyPredeclared}},
	"byte":       {t: uint8TypeInfo},
	"bool":       {t: &typeInfo{Type: boolType, Properties: propertyIsType | propertyPredeclared}},
	"complex128": {t: &typeInfo{Type: complex128Type, Properties: propertyIsType | propertyPredeclared}},
	"complex64":  {t: &typeInfo{Type: complex64Type, Properties: propertyIsType | propertyPredeclared}},
	"error":      {t: &typeInfo{Type: reflect.TypeOf((*error)(nil)).Elem(), Properties: propertyIsType | propertyPredeclared}},
	"float32":    {t: &typeInfo{Type: reflect.TypeOf(float32(0)), Properties: propertyIsType | propertyPredeclared}},
	"float64":    {t: &typeInfo{Type: float64Type, Properties: propertyIsType | propertyPredeclared}},
	"false":      {t: &typeInfo{Type: boolType, Properties: propertyUntyped, Constant: boolConst(false)}},
	"int":        {t: &typeInfo{Type: intType, Properties: propertyIsType | propertyPredeclared}},
	"int16":      {t: &typeInfo{Type: reflect.TypeOf(int16(0)), Properties: propertyIsType | propertyPredeclared}},
	"int32":      {t: int32TypeInfo},
	"int64":      {t: &typeInfo{Type: reflect.TypeOf(int64(0)), Properties: propertyIsType | propertyPredeclared}},
	"int8":       {t: &typeInfo{Type: reflect.TypeOf(int8(0)), Properties: propertyIsType | propertyPredeclared}},
	"rune":       {t: int32TypeInfo},
	"string":     {t: &typeInfo{Type: stringType, Properties: propertyIsType | propertyPredeclared}},
	"true":       {t: &typeInfo{Type: boolType, Properties: propertyUntyped, Constant: boolConst(true)}},
	"uint":       {t: &typeInfo{Type: uintType, Properties: propertyIsType | propertyPredeclared}},
	"uint16":     {t: &typeInfo{Type: reflect.TypeOf(uint16(0)), Properties: propertyIsType | propertyPredeclared}},
	"uint32":     {t: &typeInfo{Type: reflect.TypeOf(uint32(0)), Properties: propertyIsType | propertyPredeclared}},
	"uint64":     {t: &typeInfo{Type: reflect.TypeOf(uint64(0)), Properties: propertyIsType | propertyPredeclared}},
	"uint8":      {t: uint8TypeInfo},
	"uintptr":    {t: &typeInfo{Type: reflect.TypeOf(uintptr(0)), Properties: propertyIsType | propertyPredeclared}},
}

type scopeVariable struct {
	ident      string
	scopeLevel int
	node       ast.Node
}

// showMacroIgnoredTi is the TypeInfo of a ShowMacro identifier which is
// undefined but has been marked as to be ignored or "todo".
var showMacroIgnoredTi = &typeInfo{}

// checkIdentifier checks an identifier. If using, ident is marked as "used".
func (tc *typechecker) checkIdentifier(ident *ast.Identifier, using bool) *typeInfo {

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

	ti, found := tc.lookupScopes(ident.Name, false)

	// Check if the identifier is the builtin 'iota'.
	if found && ti == universe["iota"].t {
		// Check if iota is defined in the current expression evaluation.
		if tc.iota >= 0 {
			return &typeInfo{
				Constant:   int64Const(tc.iota),
				Type:       intType,
				Properties: propertyUntyped,
			}
		}
		// The identifier is the builtin 'iota', but 'iota' is not defined in
		// the current expression evaluation, so the identifier 'iota' is
		// undefined.
		found = false
	}

	// If identifiers is a ShowMacro identifier, first needs to check if
	// ShowMacro contains a "or ignore" or "or todo" option. In such cases,
	// error should not be returned, and function call should be removed
	// from tree.
	if !found {
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

	if ti.IsBuiltinFunction() {
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
			if d == ident.Name {
				delete(tc.unusedImports, pkg)
				break unusedLoop
			}
		}
	}

	tc.typeInfos[ident] = ti
	return ti
}

// checkArrayType checks an array type. If the array node is used in a composite
// literal and has no length (has ellipses), the length parameters is used
// instead.
func (tc *typechecker) checkArrayType(array *ast.ArrayType, length int) *typeInfo {
	elem := tc.checkType(array.ElementType)
	if array.Len == nil { // ellipsis.
		if length == -1 {
			panic(tc.errorf(array, "use of [...] array outside of array literal"))
		}
		tc.typeInfos[array] = &typeInfo{Properties: propertyIsType, Type: tc.types.ArrayOf(length, elem.Type)}
		return tc.typeInfos[array]
	}
	len := tc.checkExpr(array.Len)
	if !len.IsConstant() {
		panic(tc.errorf(array, "non-constant array bound %s", array.Len))
	}
	c, err := len.Constant.representedBy(intType)
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
	tc.typeInfos[array] = &typeInfo{Properties: propertyIsType, Type: tc.types.ArrayOf(b, elem.Type)}
	return tc.typeInfos[array]
}

// checkExpr type checks an expression and returns its type info.
func (tc *typechecker) checkExpr(expr ast.Expression) *typeInfo {
	ti := tc.typeof(expr, false)
	if ti.IsType() {
		panic(tc.errorf(expr, "type %s is not an expression", ti))
	}
	tc.typeInfos[expr] = ti
	return ti
}

// checkType type checks a type and returns its type info.
func (tc *typechecker) checkType(expr ast.Expression) *typeInfo {
	ti := tc.typeof(expr, true)
	if !ti.IsType() {
		panic(tc.errorf(expr, "%s is not a type", expr))
	}
	tc.typeInfos[expr] = ti
	return ti
}

// checkExprOrType type checks an expression or a type and returns its type
// info.
func (tc *typechecker) checkExprOrType(expr ast.Expression) *typeInfo {
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
func (tc *typechecker) typeof(expr ast.Expression, typeExpected bool) *typeInfo {

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
		return &typeInfo{
			Type:       typ,
			Properties: propertyUntyped,
			Constant:   c,
		}

	case *ast.UnaryOperator:
		// Handle 'not a' expressions.
		if expr.Op == ast.OperatorRelaxedNot {
			ti := tc.checkExpr(expr.Expr)
			// Non-boolean constant expressions are not allowed as operand of
			// 'not' operator.
			if ti.IsConstant() && ti.Type.Kind() != reflect.Bool {
				panic(tc.errorf(expr.Expr, "non-bool constant %s not allowed with operator not", expr.Expr))
			}
			// If the operand has kind bool then the operators is replaced with
			// '!', else the operator is replaced with an unary operator that
			// returns true only if the value is the zero of its type.
			if ti.Type.Kind() == reflect.Bool {
				expr.Op = ast.OperatorNot
			} else {
				expr.Op = internalOperatorZero
			}
		}
		// Handle the 'zero' and the 'notZero' internal operators.
		if expr.Op == internalOperatorZero || expr.Op == internalOperatorNotZero {
			ti := tc.checkExpr(expr.Expr)
			ti.setValue(nil)
			return &typeInfo{
				Type:       boolType,
				Properties: propertyUntyped,
			}
		}
		t := tc.checkExprOrType(expr.Expr)
		if t.IsType() {
			if expr.Op == ast.OperatorPointer {
				return &typeInfo{Properties: propertyIsType, Type: tc.types.PtrTo(t.Type)}
			}
			panic(tc.errorf(expr, "type %s is not an expression", t))
		}
		ti := &typeInfo{
			Type:       t.Type,
			Properties: t.Properties & propertyUntyped,
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
				ti.Constant, _ = t.Constant.unaryOp(ast.OperatorNot, nil)
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
				ti.Constant, _ = t.Constant.unaryOp(ast.OperatorSubtraction, nil)
				if !t.Untyped() {
					if _, err := ti.Constant.representedBy(ti.Type); err != nil {
						panic(tc.errorf(expr, "%s", err))
					}
				}
			}
		case ast.OperatorPointer:
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
			ti.Properties = ti.Properties | propertyAddressable
		case ast.OperatorAddress:
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
				ti.Constant, _ = t.Constant.unaryOp(ast.OperatorXor, t.Type)
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

		// Handle 'a and b' and 'a or b' expressions.
		if expr.Op == ast.OperatorRelaxedAnd || expr.Op == ast.OperatorRelaxedOr {
			t1 := tc.checkExpr(expr.Expr1)
			t2 := tc.checkExpr(expr.Expr2)
			// Non-boolean constant expressions are not allowed in left or right
			// side of the 'and' and 'or' operator.
			if t1.IsConstant() && t1.Type.Kind() != reflect.Bool {
				panic(tc.errorf(expr.Expr1, "non-bool constant %s not allowed with operator %s", expr.Expr1, expr.Op))
			}
			if t2.IsConstant() && t2.Type.Kind() != reflect.Bool {
				panic(tc.errorf(expr.Expr2, "non-bool constant %s not allowed with operator %s", expr.Expr2, expr.Op))
			}
			// Replace the non-boolean expressions with an unary operator that
			// returns true only if the value is not the zero of its type.
			if t1.Type.Kind() != reflect.Bool {
				expr.Expr1 = ast.NewUnaryOperator(expr.Expr1.Pos(), internalOperatorNotZero, expr.Expr1)
			}
			if t2.Type.Kind() != reflect.Bool {
				expr.Expr2 = ast.NewUnaryOperator(expr.Expr2.Pos(), internalOperatorNotZero, expr.Expr2)
			}
			// Change the 'and' and 'or' operators to '&&' and '||', because the
			// two expressions are now both booleans.
			if expr.Op == ast.OperatorRelaxedAnd {
				expr.Op = ast.OperatorAnd
			} else {
				expr.Op = ast.OperatorOr
			}
		}

		t, err := tc.binaryOp(expr.Expr1, expr.Op, expr.Expr2)
		if err != nil {
			if err == errDivisionByZero {
				panic(tc.errorf(expr, "%s", err))
			}
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
			if fd.Idents == nil {
				// Not implemented: see https://github.com/open2b/scriggo/issues/367
			} else {
				// Explicit field declaration.
				for _, ident := range fd.Idents {
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
					for _, field := range fields {
						if field.Name == name {
							panic(tc.errorf(ident, "duplicate field %s", ident.Name))
						}
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
		return &typeInfo{
			Type:       t,
			Properties: propertyIsType,
		}

	case *ast.MapType:
		key := tc.checkType(expr.KeyType)
		value := tc.checkType(expr.ValueType)
		defer func() {
			if rec := recover(); rec != nil {
				panic(tc.errorf(expr, "invalid map key type %s", key))
			}
		}()
		return &typeInfo{Properties: propertyIsType, Type: tc.types.MapOf(key.Type, value.Type)}

	case *ast.SliceType:
		elem := tc.checkType(expr.ElementType)
		return &typeInfo{Properties: propertyIsType, Type: tc.types.SliceOf(elem.Type)}

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
		return &typeInfo{Properties: propertyIsType, Type: tc.types.ChanOf(dir, elem.Type)}

	case *ast.CompositeLiteral:
		return tc.checkCompositeLiteral(expr, nil)

	case *ast.Interface:
		return &typeInfo{Type: emptyInterfaceType, Properties: propertyIsType | propertyPredeclared}

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
		return &typeInfo{Type: expr.Reflect, Properties: propertyIsType}

	case *ast.Func:
		tc.enterScope()
		t := tc.checkType(expr.Type)
		expr.Type.Reflect = t.Type
		tc.addToAncestors(expr)
		// Adds parameters to the function body scope.
		tc.checkDuplicateParams(expr.Type)
		tc.addMissingTypes(expr.Type)
		isVariadic := expr.Type.IsVariadic
		for i, f := range expr.Type.Parameters {
			t := tc.checkType(f.Type)
			if f.Ident != nil && !isBlankIdentifier(f.Ident) {
				if isVariadic && i == len(expr.Type.Parameters)-1 {
					tc.assignScope(f.Ident.Name, &typeInfo{Type: tc.types.SliceOf(t.Type), Properties: propertyAddressable}, f.Ident)
					continue
				}
				tc.assignScope(f.Ident.Name, &typeInfo{Type: t.Type, Properties: propertyAddressable}, f.Ident)
			}
		}
		// Adds named return values to the function body scope.
		for _, f := range expr.Type.Result {
			t := tc.checkType(f.Type)
			if f.Ident != nil && !isBlankIdentifier(f.Ident) {
				tc.assignScope(f.Ident.Name, &typeInfo{Type: t.Type, Properties: propertyAddressable}, f.Ident)
			}
		}
		expr.Body.Nodes = tc.checkNodes(expr.Body.Nodes)
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
		return &typeInfo{Type: t.Type}

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
			return &typeInfo{Type: uint8Type}
		case reflect.Slice:
			tc.checkIndex(expr.Index, t, false)
			return &typeInfo{Type: t.Type.Elem(), Properties: propertyAddressable}
		case reflect.Array:
			tc.checkIndex(expr.Index, t, false)
			ti := &typeInfo{Type: t.Type.Elem()}
			if t.Addressable() {
				ti.Properties = propertyAddressable
			}
			return ti
		case reflect.Ptr:
			elemType := t.Type.Elem()
			if elemType.Kind() != reflect.Array {
				panic(tc.errorf(expr, "invalid operation: %v (type %s does not support indexing)", expr, t))
			}
			tc.checkIndex(expr.Index, t, false)
			// Transform pa[i] to (*pa)[i].
			unOp := ast.NewUnaryOperator(expr.Expr.Pos(), ast.OperatorPointer, expr.Expr)
			tc.typeInfos[unOp] = &typeInfo{Type: elemType}
			expr.Expr = unOp
			return &typeInfo{Type: elemType.Elem(), Properties: propertyAddressable}
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
			return &typeInfo{Type: t.Type.Elem()}
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
			unOp := ast.NewUnaryOperator(expr.Expr.Pos(), ast.OperatorPointer, expr.Expr)
			tc.typeInfos[unOp] = &typeInfo{
				Type: t.Type.Elem(),
			}
			expr.Expr = unOp
		}
		switch kind {
		case reflect.String, reflect.Slice:
			return &typeInfo{Type: t.Type}
		case reflect.Array, reflect.Ptr:
			return &typeInfo{Type: tc.types.SliceOf(realType.Elem())}
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
					pkg := ti.value.(*packageInfo)
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
		if expr.Ident == "_" {
			panic(tc.errorf(expr, "cannot refer to blank field or method"))
		}
		method, trans, ok := tc.methodByName(t, expr.Ident)
		if ok {
			switch trans {
			case receiverAddAddress:
				if t.Addressable() {
					elem, _ := tc.lookupScopesElem(expr.Expr.(*ast.Identifier).Name, false)
					tc.indirectVars[elem.decl] = true
					expr.Expr = ast.NewUnaryOperator(expr.Pos(), ast.OperatorAddress, expr.Expr)
					tc.typeInfos[expr.Expr] = &typeInfo{
						Type:       tc.types.PtrTo(t.Type),
						MethodType: t.MethodType,
					}
				}
			case receiverAddIndirect:
				expr.Expr = ast.NewUnaryOperator(expr.Pos(), ast.OperatorPointer, expr.Expr)
				tc.typeInfos[expr.Expr] = &typeInfo{
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
				unOp := ast.NewUnaryOperator(expr.Expr.Pos(), ast.OperatorPointer, expr.Expr)
				tc.typeInfos[unOp] = &typeInfo{
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
		return &typeInfo{
			Type:       typ.Type,
			Properties: t.Properties & propertyAddressable,
		}

	}

	panic(fmt.Errorf("unexpected: %v (type %T)", expr, expr))
}

// checkIndex checks the type of expr as an index in a index or slice
// expression. If it is a constant returns the integer value, otherwise
// returns -1.
func (tc *typechecker) checkIndex(expr ast.Expression, t *typeInfo, isSlice bool) constant {
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
		c, err := index.Constant.representedBy(intType)
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
func (tc *typechecker) binaryOp(expr1 ast.Expression, op ast.OperatorType, expr2 ast.Expression) (*typeInfo, error) {

	t1 := tc.checkExpr(expr1)
	t2 := tc.checkExpr(expr2)

	isShift := op == ast.OperatorLeftShift || op == ast.OperatorRightShift
	isUntyped := t1.Untyped() && t2.Untyped()

	if isShift {
		if !(t1.Untyped() && t1.IsNumeric() || !t1.Untyped() && t1.IsInteger()) {
			return nil, fmt.Errorf("shift of type %s", t1)
		}
		if t2.Nil() {
			return nil, errors.New("cannot convert nil to type uint")
		}
		if !(t2.Untyped() && t2.IsNumeric() || !t2.Untyped() && t2.IsInteger()) {
			return nil, fmt.Errorf("shift count type %s, must be integer", t2.ShortString())
		}
	} else if t1.Untyped() != t2.Untyped() {
		// Make both typed.
		if t1.Untyped() {
			c, err := tc.convert(t1, expr1, t2.Type)
			if err != nil {
				if err == errNotRepresentable {
					err = fmt.Errorf("cannot convert %v (type %s) to type %s", t1.Constant, t1, t2)
				}
				return nil, err
			}
			if t1.Nil() {
				if op != ast.OperatorEqual && op != ast.OperatorNotEqual {
					return nil, fmt.Errorf("operator %s not defined on %s", op, t2.Type.Kind())
				}
				return untypedBoolTypeInfo, nil
			}
			t1.setValue(t2.Type)
			t1 = &typeInfo{Type: t2.Type, Constant: c}
		} else {
			c, err := tc.convert(t2, expr2, t1.Type)
			if err != nil {
				if err == errNotRepresentable {
					err = fmt.Errorf("cannot convert %v (type %s) to type %s", t2.Constant, t2, t1)
				}
				return nil, err
			}
			if t2.Nil() {
				if op != ast.OperatorEqual && op != ast.OperatorNotEqual {
					return nil, fmt.Errorf("operator %s not defined on %s", op, t1.Type.Kind())
				}
				return untypedBoolTypeInfo, nil
			}
			t2.setValue(t1.Type)
			t2 = &typeInfo{Type: t1.Type, Constant: c}
		}
	}

	if t1.IsConstant() && t2.IsConstant() {

		if isShift {
			t1.setValue(nil)
			t2.setValue(nil)
		} else {
			if t1.Type != t2.Type && !(t1.Untyped() && t1.IsNumeric() && t2.IsNumeric()) {
				return nil, fmt.Errorf("mismatched types %s and %s", t1.ShortString(), t2.ShortString())
			}
		}

		c, err := t1.Constant.binaryOp(op, t2.Constant)
		if err != nil {
			switch err {
			case errInvalidOperation:
				if op == ast.OperatorModulo && isUntyped {
					if isComplex(t1.Type.Kind()) {
						err = fmt.Errorf("operator %% not defined on untyped complex")
					} else {
						err = errors.New("floating-point % operation")
					}
				} else {
					err = fmt.Errorf("operator %s not defined on %s", op, t1.ShortString())
				}
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

		if !t1.Untyped() && !evalToBoolOperators[op] {
			c, err = c.representedBy(t1.Type)
			if err != nil {
				return nil, err
			}
		}

		typ := t1.Type
		if evalToBoolOperators[op] {
			typ = boolType
		} else if t1.Untyped() && t1.Type.Kind() < t2.Type.Kind() {
			typ = t2.Type
		}
		ti := &typeInfo{Type: typ, Constant: c}
		if t1.Untyped() || isComparison(op) {
			ti.Properties = propertyUntyped
		}

		return ti, nil
	}

	if isShift {
		var err error
		if t2.IsConstant() {
			_, err = t2.Constant.representedBy(uintType)
			if err != nil {
				return nil, err
			}
			t2.setValue(uintType)
		}
		ti := &typeInfo{Type: t1.Type}
		if t1.Untyped() {
			ti.Properties = propertyUntyped
			ti.Type = t1.Type
		}
		t1.setValue(nil)
		t2.setValue(nil)
		return ti, nil
	}

	if isUntyped {
		if t1.Nil() && t2.Nil() {
			return nil, fmt.Errorf("operator %s not defined on nil", op)
		}
		if t1.Nil() {
			return nil, fmt.Errorf("cannot convert nil to type %s", t2.Type)
		}
		if t2.Nil() {
			return nil, fmt.Errorf("cannot convert nil to type %s", t1.Type)
		}
		k1, k2 := t1.Type.Kind(), t2.Type.Kind()
		if !(k1 == k2 || isNumeric(k1) && isNumeric(k2)) {
			return nil, fmt.Errorf("mismatched types %s and %s", t1.Type, t2.Type)
		}
		typ := t1.Type
		switch {
		case isBoolean(k1):
			if !operatorsOfKind[k1][op] {
				return nil, fmt.Errorf("operator %s not defined on bool", op)
			}
		case isNumeric(k1):
			if k1 < k2 {
				typ = t2.Type
			}
			if isComparison(op) {
				_, err := tc.convert(t1, expr1, typ)
				if err != nil {
					panic(tc.errorf(expr1, "%s", err))
				}
				_, err = tc.convert(t2, expr2, typ)
				if err != nil {
					panic(tc.errorf(expr2, "%s", err))
				}
			}
		}
		t1.setValue(typ)
		t2.setValue(typ)
		if isComparison(op) {
			typ = boolType
		}
		return &typeInfo{Type: typ, Properties: propertyUntyped}, nil
	}

	t1.setValue(nil)
	t2.setValue(nil)

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
		return untypedBoolTypeInfo, nil
	}

	if t1.Type != t2.Type {
		return nil, fmt.Errorf("mismatched types %s and %s", t1.ShortString(), t2.ShortString())
	}

	if kind := t1.Type.Kind(); !operatorsOfKind[kind][op] {
		return nil, fmt.Errorf("operator %s not defined on %s", op, kind)
	}

	if (op == ast.OperatorDivision || op == ast.OperatorModulo) && t2.IsConstant() && t2.Constant.zero() {
		return nil, errDivisionByZero
	}

	return &typeInfo{Type: t1.Type}, nil
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
		c, err := size.Constant.representedBy(intType)
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
func (tc *typechecker) checkBuiltinCall(expr *ast.Call) []*typeInfo {

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
			for _, el := range expr.Args[1:] {
				t := tc.checkExpr(el)
				if err := tc.isAssignableTo(t, el, elemType); err != nil {
					switch err.(type) {
					case invalidTypeInAssignment:
						panic(tc.errorf(expr, "%s in append", err))
					case nilConvertionError:
						panic(tc.errorf(expr, "cannot use nil as type %s in append", elemType))
					default:
						panic(tc.errorf(expr, "%s", err))
					}
				}
				t.setValue(elemType)
			}
		}
		return []*typeInfo{{Type: slice.Type}}

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
		ti := &typeInfo{Type: intType}
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
		return []*typeInfo{ti}

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
		return []*typeInfo{}

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
			ti := &typeInfo{
				Type:       complex128Type,
				Constant:   newComplexConst(re.Constant, im.Constant),
				Properties: propertyUntyped,
			}
			return []*typeInfo{ti}
		}
		if re.IsUntypedConstant() {
			c, err := tc.convert(re, expr.Args[0], im.Type)
			if err != nil {
				panic(tc.errorf(expr, "cannot convert %s (type %s) to type %s", re.Constant, re, im))
			}
			re.setValue(im.Type)
			re = &typeInfo{Type: im.Type, Constant: c}
		} else if im.IsUntypedConstant() {
			c, err := tc.convert(im, expr.Args[1], re.Type)
			if err != nil {
				panic(tc.errorf(expr, "cannot convert %s (type %s) to type %s", im.Constant, im, re))
			}
			im.setValue(re.Type)
			im = &typeInfo{Type: re.Type, Constant: c}
		} else if reKind != imKind {
			panic(tc.errorf(expr, "invalid operation: %s (mismatched types %s and %s)", expr, re, im))
		}
		ti := &typeInfo{}
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
		} else {
			im.setValue(nil)
			re.setValue(nil)
		}
		return []*typeInfo{ti}

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
		src.setValue(nil)
		return []*typeInfo{{Type: intType}}

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
			_, err := tc.convert(key, expr.Args[1], keyType)
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
		ti := &typeInfo{Type: intType}
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
		return []*typeInfo{ti}

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
		return []*typeInfo{{Type: t.Type}}

	case "new":
		if len(expr.Args) == 0 {
			panic(tc.errorf(expr, "missing argument to new"))
		}
		t := tc.checkType(expr.Args[0])
		if len(expr.Args) > 1 {
			panic(tc.errorf(expr, "too many arguments to new(%s)", expr.Args[0]))
		}
		return []*typeInfo{{Type: tc.types.PtrTo(t.Type)}}

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
			if err := tc.isAssignableTo(ti, expr.Args[0], emptyInterfaceType); err != nil {
				panic(tc.errorf(expr, "%s", err))
			}
			ti.setValue(nil)
		}
		return nil

	case "print", "println":
		for _, arg := range expr.Args {
			ti := tc.checkExpr(arg)
			if ti.Nil() {
				panic(tc.errorf(expr, "use of untyped nil"))
			}
			if err := tc.isAssignableTo(ti, arg, emptyInterfaceType); err != nil {
				panic(tc.errorf(expr, "%s", err))
			}
			ti.setValue(nil)
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
		ti := &typeInfo{Type: float64Type}
		if t.IsUntypedConstant() {
			if !isNumeric(t.Type.Kind()) {
				panic(tc.errorf(expr, "invalid argument %s (type %s) for %s", expr.Args[0], t, ident.Name))
			}
			ti.Properties = propertyUntyped
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
		return []*typeInfo{ti}

	case "recover":
		if len(expr.Args) > 0 {
			panic(tc.errorf(expr, "too many arguments to recover"))
		}
		return []*typeInfo{{Type: emptyInterfaceType}}

	}

	panic(fmt.Sprintf("unexpected builtin %s", ident.Name))

}

// checkCallExpression type checks a call expression, including type
// conversions and built-in function calls. Returns a list of typeinfos
// obtained from the call and returns two booleans indicating respectively if
// expr is a builtin call or a conversion.
func (tc *typechecker) checkCallExpression(expr *ast.Call, statement bool) ([]*typeInfo, bool, bool) {

	// Check a builtin function call.
	if ident, ok := expr.Func.(*ast.Identifier); ok {
		if ti, ok := tc.lookupScopes(ident.Name, false); ok && ti.IsBuiltinFunction() {
			tc.typeInfos[expr.Func] = ti
			return tc.checkBuiltinCall(expr), true, false
		}
	}

	t := tc.checkExprOrType(expr.Func)

	switch t.MethodType {
	case methodValueConcrete:
		t.MethodType = methodCallConcrete
	case methodValueInterface:
		t.MethodType = methodCallInterface
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
		ti := tc.checkExplicitConversion(expr)
		return []*typeInfo{ti}, false, true
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
	resultTypes := make([]*typeInfo, numOut)
	for i := 0; i < numOut; i++ {
		resultTypes[i] = &typeInfo{Type: t.Type.Out(i)}
	}

	return resultTypes, false, false
}

func (tc *typechecker) checkExplicitConversion(expr *ast.Call) *typeInfo {

	t := tc.typeInfos[expr.Func]

	if len(expr.Args) == 0 {
		panic(tc.errorf(expr, "missing argument to conversion to %s: %s", t, expr))
	}
	if len(expr.Args) > 1 {
		panic(tc.errorf(expr, "too many arguments to conversion to %s: %s", t, expr))
	}

	arg := tc.checkExpr(expr.Args[0])

	var c constant
	var err error

	switch {
	case arg.IsConstant():
		k := t.Type.Kind()
		if k == reflect.Interface {
			if tc.emptyMethodSet(t.Type) {
				_, err = arg.Constant.representedBy(arg.Type)
			} else {
				err = errTypeConversion
			}
		} else {
			argKind := arg.Type.Kind()
			switch {
			case k == reflect.String && isInteger(argKind):
				// As a special case, an integer constant can be explicitly
				// converted to a string type.
				n, _ := arg.Constant.representedBy(int64Type)
				if n == nil {
					c = stringConst("\uFFFD")
				} else {
					c = stringConst(n.int64())
				}
			case k == reflect.Slice && argKind == reflect.String:
				// As a special case, a string constant can be explicitly converted
				// to a slice of runes or bytes.
				if elem := t.Type.Elem(); elem != uint8Type && elem != int32Type {
					err = errTypeConversion
				}
			default:
				c, err = representedBy(arg, t.Type)
			}
		}
	case arg.Untyped():
		_, err = tc.convert(arg, expr.Args[0], t.Type)
	default:
		if !tc.types.ConvertibleTo(arg.Type, t.Type) {
			err = errTypeConversion
		}
	}

	if err != nil {
		if err == errTypeConversion {
			panic(tc.errorf(expr, "cannot convert %s (type %s) to type %s", expr.Args[0], arg.Type, t.Type))
		}
		panic(tc.errorf(expr, "%s", err))
	}

	if arg.Nil() {
		return tc.nilOf(t.Type)
	}
	ti := &typeInfo{Type: t.Type, Constant: c}
	if c == nil {
		arg.setValue(arg.Type)
	} else {
		ti.setValue(t.Type)
	}

	return ti
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
func (tc *typechecker) checkCompositeLiteral(node *ast.CompositeLiteral, typ reflect.Type) *typeInfo {

	// Handle composite literal nodes with implicit type.
	if node.Type == nil {
		node.Type = ast.NewPlaceholder()
		tc.typeInfos[node.Type] = &typeInfo{Properties: propertyIsType, Type: typ}
	}

	maxIndex := -1
	switch node.Type.(type) {
	case *ast.ArrayType, *ast.SliceType:
		maxIndex = tc.maxIndex(node)
	}

	// Check the type of the composite literal. Arrays are handled in a special
	// way since composite literals support array types declarations with
	// ellipses as length.
	var ti *typeInfo
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
				ti := &typeInfo{Type: ti.Type}
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
				if !isExported(fieldTi.Name) && !strings.HasPrefix(fieldTi.Name, "𝗽") {
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
			var elemTi *typeInfo
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
			var elemTi *typeInfo
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
			var keyTi *typeInfo
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
			var valueTi *typeInfo
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

	default:

		panic(tc.errorf(node, "invalid type for composite literal: %s", ti))

	}

	nodeTi := &typeInfo{Type: ti.Type}
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
