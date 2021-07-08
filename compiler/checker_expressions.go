// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/open2b/scriggo/compiler/ast"
	"github.com/open2b/scriggo/runtime"
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

var uint8TypeInfo = &typeInfo{Type: uint8Type, Properties: propertyIsType | propertyUniverse}
var int32TypeInfo = &typeInfo{Type: int32Type, Properties: propertyIsType | propertyUniverse}

var untypedBoolTypeInfo = &typeInfo{Type: boolType, Properties: propertyUntyped}

var errorType = reflect.TypeOf((*error)(nil)).Elem()

var envType = reflect.TypeOf((*runtime.Env)(nil)).Elem()

var universe = typeCheckerScope{
	"append":     {t: &typeInfo{Properties: propertyUniverse}},
	"cap":        {t: &typeInfo{Properties: propertyUniverse}},
	"close":      {t: &typeInfo{Properties: propertyUniverse}},
	"complex":    {t: &typeInfo{Properties: propertyUniverse}},
	"copy":       {t: &typeInfo{Properties: propertyUniverse}},
	"delete":     {t: &typeInfo{Properties: propertyUniverse}},
	"imag":       {t: &typeInfo{Properties: propertyUniverse}},
	"iota":       {t: &typeInfo{Properties: propertyUniverse, Type: intType}},
	"len":        {t: &typeInfo{Properties: propertyUniverse}},
	"make":       {t: &typeInfo{Properties: propertyUniverse}},
	"new":        {t: &typeInfo{Properties: propertyUniverse}},
	"nil":        {t: &typeInfo{Properties: propertyUntyped | propertyUniverse}},
	"panic":      {t: &typeInfo{Properties: propertyUniverse}},
	"print":      {t: &typeInfo{Properties: propertyUniverse}},
	"println":    {t: &typeInfo{Properties: propertyUniverse}},
	"real":       {t: &typeInfo{Properties: propertyUniverse}},
	"recover":    {t: &typeInfo{Properties: propertyUniverse}},
	"byte":       {t: uint8TypeInfo},
	"bool":       {t: &typeInfo{Type: boolType, Properties: propertyIsType | propertyUniverse}},
	"complex128": {t: &typeInfo{Type: complex128Type, Properties: propertyIsType | propertyUniverse}},
	"complex64":  {t: &typeInfo{Type: complex64Type, Properties: propertyIsType | propertyUniverse}},
	"error":      {t: &typeInfo{Type: errorType, Properties: propertyIsType | propertyUniverse}},
	"float32":    {t: &typeInfo{Type: reflect.TypeOf(float32(0)), Properties: propertyIsType | propertyUniverse}},
	"float64":    {t: &typeInfo{Type: float64Type, Properties: propertyIsType | propertyUniverse}},
	"false":      {t: &typeInfo{Type: boolType, Properties: propertyUniverse | propertyUntyped, Constant: boolConst(false)}},
	"int":        {t: &typeInfo{Type: intType, Properties: propertyIsType | propertyUniverse}},
	"int16":      {t: &typeInfo{Type: reflect.TypeOf(int16(0)), Properties: propertyIsType | propertyUniverse}},
	"int32":      {t: int32TypeInfo},
	"int64":      {t: &typeInfo{Type: reflect.TypeOf(int64(0)), Properties: propertyIsType | propertyUniverse}},
	"int8":       {t: &typeInfo{Type: reflect.TypeOf(int8(0)), Properties: propertyIsType | propertyUniverse}},
	"rune":       {t: int32TypeInfo},
	"string":     {t: &typeInfo{Type: stringType, Properties: propertyIsType | propertyIsFormatType | propertyUniverse}},
	"this":       {t: &typeInfo{Properties: propertyUniverse | propertyUntyped | propertyAddressable}},
	"true":       {t: &typeInfo{Type: boolType, Properties: propertyUniverse | propertyUntyped, Constant: boolConst(true)}},
	"uint":       {t: &typeInfo{Type: uintType, Properties: propertyIsType | propertyUniverse}},
	"uint16":     {t: &typeInfo{Type: reflect.TypeOf(uint16(0)), Properties: propertyIsType | propertyUniverse}},
	"uint32":     {t: &typeInfo{Type: reflect.TypeOf(uint32(0)), Properties: propertyIsType | propertyUniverse}},
	"uint64":     {t: &typeInfo{Type: reflect.TypeOf(uint64(0)), Properties: propertyIsType | propertyUniverse}},
	"uint8":      {t: uint8TypeInfo},
	"uintptr":    {t: &typeInfo{Type: reflect.TypeOf(uintptr(0)), Properties: propertyIsType | propertyUniverse}},
}

type scopeVariable struct {
	ident      string
	scopeLevel int
	node       ast.Node
}

// checkIdentifier checks an identifier. If used, ident is marked as "used".
func (tc *typechecker) checkIdentifier(ident *ast.Identifier, used bool) *typeInfo {

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

	// Handle the predeclared identifier 'this' when checking the 'using'
	// statement.
	if found && ti == universe["this"].t {
		if tc.using.withinUsingStmt {
			thisCurrName := tc.thisCurrentName()
			ident.Name = thisCurrName
			usingData := tc.compilation.thisToUsingData[thisCurrName]
			usingData.used = true
			if tc.toBeEmitted {
				usingData.toBeEmitted = true
			}
			tc.compilation.thisToUsingData[thisCurrName] = usingData
			ti, _ = tc.lookupScopes(ident.Name, false)
		} else {
			// The identifier is the predeclared identifier 'this', but 'this'
			// is not defined outside an 'using' statement so it is considered
			// undefined.
			found = false
		}
	}

	if !found {
		panic(tc.errorf(ident, "undefined: %s", ident.Name))
	}

	if ti.IsBuiltinFunction() {
		panic(tc.errorf(ident, "use of builtin %s not in function call", ident.Name))
	}

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

	// Handle predeclared variables in templates and scripts.
	if tc.opts.mod == templateMod || tc.opts.mod == scriptMod {
		// The identifier refers to a predefined value that is an up value for
		// the current function.
		if ti.IsPredefined() && tc.isUpVar(ident.Name) {
			// The type info contains a *reflect.Value, so it is a variable.
			if rv, ok := ti.value.(*reflect.Value); ok {
				// Get the list of the nested functions, from the outermost to
				// the innermost (which is the function that refers to the
				// identifier).
				if nestedFuncs := tc.nestedFuncs(); len(nestedFuncs) > 0 {
					upvar := ast.Upvar{
						PredefinedName:      ident.Name,
						PredefinedPkg:       ident.Name,
						PredefinedValue:     rv,
						PredefinedValueType: ti.Type,
						Index:               -1,
					}
					for _, fn := range nestedFuncs {
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
			}
		}
	}

	// Mark identifier as "used".
	if used {
		for i := len(tc.unusedVars) - 1; i >= 0; i-- {
			v := tc.unusedVars[i]
			if v.ident == ident.Name {
				v.node = nil
				break
			}
		}
	}

	// Mark 'this' as used, when 'using' is a package-level statement.
	if strings.HasPrefix(ident.Name, "$this") {
		ud := tc.compilation.thisToUsingData[ident.Name]
		ud.used = true
		if tc.toBeEmitted {
			ud.toBeEmitted = true
		}
		tc.compilation.thisToUsingData[ident.Name] = ud
	}

	// For "." imported packages, mark package as used.
	for pkg, imp := range tc.unusedImports {
		if imp.decl[ident.Name] == ti {
			delete(tc.unusedImports, pkg)
			break
		}
	}

	tc.compilation.typeInfos[ident] = ti
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
		tc.compilation.typeInfos[array] = &typeInfo{Properties: propertyIsType, Type: tc.types.ArrayOf(length, elem.Type)}
		return tc.compilation.typeInfos[array]
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
	tc.compilation.typeInfos[array] = &typeInfo{Properties: propertyIsType, Type: tc.types.ArrayOf(b, elem.Type)}
	return tc.compilation.typeInfos[array]
}

// checkExpr type checks an expression and returns its type info.
func (tc *typechecker) checkExpr(expr ast.Expression) *typeInfo {
	ti := tc.typeof(expr, false)
	if ti.IsType() {
		panic(tc.errorf(expr, "type %s is not an expression", ti))
	}
	tc.compilation.typeInfos[expr] = ti
	return ti
}

// checkExpr2 type checks an expression and returns a pair of type infos. If
// the expression is a default expression it returns the type infos of the
// left and right expressions. If the left expression it is not resolved or
// expr is not a default expression the first type info is nil.
// show indicates if the expression is in a show statement.
func (tc *typechecker) checkExpr2(expr ast.Expression, show bool) typeInfoPair {
	if expr, ok := expr.(*ast.Default); ok {
		return tc.checkDefault(expr, show)
	}
	return typeInfoPair{nil, tc.checkExpr(expr)}
}

// subExpr returns a sub-expression of the expression expr if it is a default
// expression, otherwise returns expr. left indicates if the left expression
// should be returned, it returns the right expression if it is false.
func subExpr(expr ast.Expression, left bool) ast.Expression {
	if expr, ok := expr.(*ast.Default); ok {
		if left {
			return expr.Expr1
		}
		return expr.Expr2
	}
	return expr
}

// checkType type checks a type and returns its type info.
func (tc *typechecker) checkType(expr ast.Expression) *typeInfo {
	ti := tc.typeof(expr, true)
	if !ti.IsType() {
		panic(tc.errorf(expr, "%s is not a type", expr))
	}
	tc.compilation.typeInfos[expr] = ti
	return ti
}

// checkExprOrType type checks an expression or a type and returns its type
// info.
func (tc *typechecker) checkExprOrType(expr ast.Expression) *typeInfo {
	ti := tc.typeof(expr, false)
	tc.compilation.typeInfos[expr] = ti
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

	ti := tc.compilation.typeInfos[expr]
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
							tc.compilation.indirectVars[tc.scopes[i][n].decl] = true
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
				if call, ok := expr.Expr.(*ast.Call); ok && tc.compilation.typeInfos[call.Func].IsBuiltinFunction() {
					s = expr.Op.String() + "(" + expr.Expr.String() + ")"
				} else {
					s = expr.String()
				}
				panic(tc.errorf(expr, "invalid operation: %s (receive from send-only type %s)", s, t.Type))
			}
			ti.Type = t.Type.Elem()
		case ast.OperatorExtendedNot:
			if t.Nil() {
				panic(tc.errorf(expr, "invalid operation: %s (operator '%s' not defined on nil)", expr, expr.Op))
			}
			if t.IsConstant() {
				return &typeInfo{
					Constant:   boolConst(t.Constant.zero()),
					Properties: propertyUntyped,
					Type:       boolType,
				}
			}
			// If the operand has kind bool then the operator is replaced with
			// '!', else the operator is replaced with an unary operator that
			// returns true only if the value is the zero of its type. In the
			// former case the syntax `not a` it's semantically equivalent to
			// `!a`.
			if t.Type.Kind() == reflect.Bool {
				expr.Op = ast.OperatorNot
			} else {
				expr.Op = internalOperatorZero
			}
			ti.Type = boolType
			ti.setValue(nil)
			return ti
		case internalOperatorZero, internalOperatorNotZero:
			ti.Type = boolType
			ti.setValue(nil)
			return ti
		}
		return ti

	case *ast.BinaryOperator:

		// Handle 'a and b' and 'a or b' expressions.
		if expr.Op == ast.OperatorExtendedAnd || expr.Op == ast.OperatorExtendedOr {
			t1 := tc.checkExpr(expr.Expr1)
			t2 := tc.checkExpr(expr.Expr2)
			if t1.Nil() || t2.Nil() {
				panic(tc.errorf(expr, "invalid operation: %s (operator '%s' not defined on nil)", expr, expr.Op))
			}
			if t1.IsConstant() && t2.IsConstant() {
				nz1 := !t1.Constant.zero()
				nz2 := !t2.Constant.zero()
				var nz bool
				if expr.Op == ast.OperatorExtendedAnd {
					nz = nz1 && nz2
				} else {
					nz = nz1 || nz2
				}
				c := &typeInfo{
					Constant:   boolConst(nz),
					Properties: propertyUntyped,
					Type:       boolType,
				}
				return c
			}
			if t1.IsConstant() {
				t1.setValue(nil)
			}
			if t2.IsConstant() {
				t2.setValue(nil)
			}
			// Replace the non-boolean expressions with an unary operator that
			// returns true only if the value is not the zero of its type.
			if t1.Type.Kind() != reflect.Bool {
				expr.Expr1 = ast.NewUnaryOperator(expr.Expr1.Pos(), internalOperatorNotZero, expr.Expr1)
			}
			if t2.Type.Kind() != reflect.Bool {
				expr.Expr2 = ast.NewUnaryOperator(expr.Expr2.Pos(), internalOperatorNotZero, expr.Expr2)
			}
			// Change the 'and' and 'or' operator to '&&' and '||', because the
			// two expressions are now both booleans.
			if expr.Op == ast.OperatorExtendedAnd {
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

	case *ast.DollarIdentifier:
		return tc.checkDollarIdentifier(expr)

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
			// If the field name is unexported, it's impossible to
			// create an new reflect.Type due to the limits that the
			// package 'reflect' currently has. The solution adopted is
			// to prepend an unicode character 𝗽 which is considered
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
			// resulting in the inability of make comparisons with that
			// types.
			//
			// Adding the unique index also is also used to check if a
			// non-exported field of a struct can be accessed from a
			// package (see the documentation of
			// typechecker.structDeclPkg).
			if fd.Idents == nil {
				// Implicit field declaration.
				name := typ.Name()
				if name == "" {
					name = typ.Elem().Name()
				}
				if fc, _ := utf8.DecodeRuneInString(name); !unicode.Is(unicode.Lu, fc) {
					name = "𝗽" + strconv.Itoa(tc.compilation.UniqueIndex(tc.path)) + name
				}
				fields = append(fields, reflect.StructField{
					Name:      name,
					Type:      typ,
					Tag:       reflect.StructTag(fd.Tag),
					Anonymous: true,
				})
			} else {
				// Explicit field declaration.
				for _, ident := range fd.Idents {
					name := ident.Name
					if fc, _ := utf8.DecodeRuneInString(name); !unicode.Is(unicode.Lu, fc) {
						name = "𝗽" + strconv.Itoa(tc.compilation.UniqueIndex(tc.path)) + ident.Name
					}
					for _, field := range fields {
						if field.Name == name {
							panic(tc.errorf(ident, "duplicate field %s", ident.Name))
						}
					}
					fields = append(fields, reflect.StructField{
						Name: name,
						Type: typ,
						Tag:  reflect.StructTag(fd.Tag),
					})
				}
			}
		}
		t := tc.types.StructOf(fields)
		tc.structDeclPkg[t] = tc.path
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

	case *ast.Default:
		panic(tc.errorf(expr, "cannot use default expression in this context"))

	case *ast.Interface:
		return &typeInfo{Type: emptyInterfaceType, Properties: propertyIsType | propertyUniverse}

	case *ast.FuncType:
		tc.checkDuplicateParams(expr)
		variadic := expr.IsVariadic
		// Parameters.
		numIn := len(expr.Parameters)
		in := make([]reflect.Type, numIn)
		for i := numIn - 1; i >= 0; i-- {
			param := expr.Parameters[i]
			if param.Type == nil {
				in[i] = in[i+1]
				param.Type = expr.Parameters[i+1].Type
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
				res.Type = expr.Result[i+1].Type
			} else {
				c := tc.checkType(res.Type)
				out[i] = c.Type
			}
		}
		if expr.Macro {
			ident := expr.Result[0].Type.(*ast.Identifier)
			if out[0] != tc.universe[ident.Name].t.Type {
				panic(tc.errorf(ident, "invalid macro result type %s", ident.Name))
			}
		}
		expr.Reflect = tc.types.FuncOf(in, out, variadic)
		return &typeInfo{Type: expr.Reflect, Properties: propertyIsType}

	case *ast.Func:
		if expr.Type.Macro && len(expr.Type.Result) == 0 {
			tc.makeMacroResultExplicit(expr)
		}
		t := tc.checkType(expr.Type)
		tc.checkFunc(expr)
		ti := &typeInfo{Type: t.Type}
		if expr.Type.Macro {
			ti.Properties |= propertyIsMacroDeclaration
		}
		return ti

	case *ast.Call:
		tis := tc.checkCallExpression(expr)
		if len(tis) == 0 {
			panic(tc.errorf(expr, "%v used as value", expr))
		}
		if len(tis) > 1 {
			panic(tc.errorf(expr, "multiple-value %v in single-value context", expr))
		}
		return tis[0]

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
			tc.compilation.typeInfos[unOp] = &typeInfo{Type: elemType}
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
				tc.compilation.typeInfos[expr.Index] = key
			} else {
				key.setValue(t.Type.Key())
			}
			return &typeInfo{Type: t.Type.Elem()}
		default:
			panic(tc.errorf(expr, "invalid operation: %s (type %s does not support indexing)", expr, t.ShortString()))
		}

	case *ast.Render:
		return tc.checkRender(expr)

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
			if t.Type != stringType && tc.isFormatType(t.Type) {
				panic(tc.errorf(expr, "invalid operation %s (slice of %s)", expr, t))
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
			tc.compilation.typeInfos[unOp] = &typeInfo{
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
		// Can be a package selector, a method expression, a method value or a field selector.
		if ident, ok := expr.Expr.(*ast.Identifier); ok {
			ti, ok := tc.lookupScopes(ident.Name, false)
			if ok && ti.IsPackage() {
				// Package selector.
				delete(tc.unusedImports, ident.Name)
				if !unicode.Is(unicode.Lu, []rune(expr.Ident)[0]) {
					panic(tc.errorf(expr, "cannot refer to unexported name %s", expr))
				}
				pkg := ti.value.(*packageInfo)
				v, ok := pkg.Declarations[expr.Ident]
				if !ok {
					panic(tc.errorf(expr, "undefined: %v", expr))
				}
				// v is a predefined variable.
				if rv, ok := v.value.(*reflect.Value); v.Addressable() && ok {
					upvar := ast.Upvar{
						PredefinedName:      expr.Ident,
						PredefinedPkg:       ident.Name,
						PredefinedValue:     rv,
						PredefinedValueType: v.Type,
						Index:               -1,
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
				tc.compilation.typeInfos[expr] = v
				return v
			}
		}
		t := tc.checkExprOrType(expr.Expr)
		if t.Nil() {
			panic(tc.errorf(expr.Expr, "use of untyped nil"))
		}
		if t.IsType() {
			// Method expression.
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
			// Method value.
			switch trans {
			case receiverAddAddress:
				if t.Addressable() {
					elem, _ := tc.lookupScopesElem(expr.Expr.(*ast.Identifier).Name, false)
					tc.compilation.indirectVars[elem.decl] = true
					expr.Expr = ast.NewUnaryOperator(expr.Pos(), ast.OperatorAddress, expr.Expr)
					tc.compilation.typeInfos[expr.Expr] = &typeInfo{
						Type:       tc.types.PtrTo(t.Type),
						MethodType: t.MethodType,
					}
				}
			case receiverAddIndirect:
				expr.Expr = ast.NewUnaryOperator(expr.Pos(), ast.OperatorPointer, expr.Expr)
				tc.compilation.typeInfos[expr.Expr] = &typeInfo{
					Type:       t.Type.Elem(),
					MethodType: t.MethodType,
				}
			}
			return method
		}
		// Field selector.
		field, newName, err := tc.fieldByName(t, expr.Ident)
		if err != nil {
			panic(tc.errorf(expr, "%v undefined (%s)", expr, err))
		}
		expr.Ident = newName
		return field

	case *ast.TypeAssertion:
		t := tc.checkExpr(expr.Expr)
		if t.Nil() {
			panic(tc.errorf(expr, "use of untyped nil"))
		}
		if t.Type.Kind() != reflect.Interface {
			panic(tc.errorf(expr, "invalid type assertion: %v (non-interface type %s on left)", expr, t))
		}
		typ := tc.checkType(expr.Type)
		if typ.Type.Kind() != reflect.Interface && !tc.types.Implements(typ.Type, t.Type) {
			panic(tc.errorf(expr, "%s", tc.errTypeAssertion(typ.Type, t.Type)))
		}
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

// binaryOp type checks the binary expression t1 op t2 and returns its type
// info. Returns an error if the operation can not be executed.
func (tc *typechecker) binaryOp(expr1 ast.Expression, op ast.OperatorType, expr2 ast.Expression) (*typeInfo, error) {

	t1 := tc.checkExpr(expr1)
	t2 := tc.checkExpr(expr2)

	var isStringContains bool
	if op == ast.OperatorContains || op == ast.OperatorNotContains {
		switch t1.Type.Kind() {
		case reflect.String:
			isStringContains = true
		case reflect.Slice, reflect.Array:
			t1 = &typeInfo{Type: t1.Type.Elem()}
		case reflect.Map:
			t1 = &typeInfo{Type: t1.Type.Key()}
		}
	}

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
	} else if isStringContains {
		if t1.Nil() || t1.Type.Kind() != reflect.String {
			return nil, fmt.Errorf("contains of type %s", t1)
		}
		if t2.Nil() {
			return nil, errors.New("cannot convert nil to type rune")
		}
		if t2.IsNumeric() {
			if !(t2.Untyped() || !t2.Untyped() && t2.IsInteger()) {
				return nil, fmt.Errorf("contains element type %s, must be integer", t2.ShortString())
			}
		} else if t2.Type.Kind() != reflect.String {
			if t2.IsConstant() {
				return nil, fmt.Errorf("cannot compare %#v (type %s) to type %s", t2.Constant, t2, t1)
			}
			return nil, fmt.Errorf("cannot compare type %s to type %s", t2, t1)
		}
	} else {
		// Return a better error message if one of the operands is not boolean.
		// See https://github.com/golang/go/commit/23573d0ea225d4b93ccd2b946b1de121c3a6cee5
		if op == ast.OperatorAnd || op == ast.OperatorOr {
			if t1.Nil() || t1.Type.Kind() != reflect.Bool {
				return nil, fmt.Errorf("operator %s not defined on %s", op, t1)
			}
			if t2.Nil() || t2.Type.Kind() != reflect.Bool {
				return nil, fmt.Errorf("operator %s not defined on %s", op, t2)
			}
		}
		if t1.Untyped() != t2.Untyped() {
			// Make both typed.
			if t1.Untyped() {
				c, err := tc.convert(t1, expr1, t2.Type)
				if err != nil {
					if err == errNotRepresentable {
						err = fmt.Errorf("cannot convert %#v (type %s) to type %s", t1.Constant, t1, t2)
					}
					return nil, err
				}
				if t1.Nil() {
					if op != ast.OperatorEqual && op != ast.OperatorNotEqual &&
						op != ast.OperatorContains && op != ast.OperatorNotContains {
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
						err = fmt.Errorf("cannot convert %#v (type %s) to type %s", t2.Constant, t2, t1)
					}
					return nil, err
				}
				if t2.Nil() {
					if op != ast.OperatorEqual && op != ast.OperatorNotEqual &&
						op != ast.OperatorContains && op != ast.OperatorNotContains {
						return nil, fmt.Errorf("operator %s not defined on %s", op, t1.Type.Kind())
					}
					return untypedBoolTypeInfo, nil
				}
				t2.setValue(t1.Type)
				t2 = &typeInfo{Type: t1.Type, Constant: c}
			}
		}
	}

	if t1.IsConstant() && t2.IsConstant() {

		if isShift || isStringContains {
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

	if isStringContains {
		t1.setValue(nil)
		if t2.IsConstant() && t2.IsNumeric() {
			_, err := t2.Constant.representedBy(runeType)
			if err != nil {
				return nil, err
			}
			t2.setValue(runeType)
		} else {
			t2.setValue(nil)
		}
		return untypedBoolTypeInfo, nil
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
		if op == ast.OperatorEqual || op == ast.OperatorNotEqual ||
			op == ast.OperatorContains || op == ast.OperatorNotContains {
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
			// Handle the special case:
			//
			//     append(t, s...)
			//
			// where 't' is a value with type 'T' (assignable to '[]byte'), and
			// 's' is a value with kind 'string'.
			//
			// In this case the expression:
			//
			//     append(t, s...)
			//
			// is transformed into:
			//
			//     append(t, T(s)...)
			//
			if isSpecialCase := t.Type.Kind() == reflect.String && slice.Type.Elem() == uint8Type; isSpecialCase {
				pos := expr.Args[1].Pos()
				T := ast.NewPlaceholder() // T
				tc.compilation.typeInfos[T] = &typeInfo{Properties: propertyIsType, Type: slice.Type}
				conversion := ast.NewCall(pos, T, []ast.Expression{expr.Args[1]}, false) // T(s)
				expr.Args[1] = conversion
				tc.checkExpr(expr.Args[1])
			} else {
				if tc.isAssignableTo(t, expr.Args[1], slice.Type) != nil {
					panic(tc.errorf(expr, "cannot use %s (type %s) as type %s in append", expr.Args[1], t, slice.Type))
				}
			}
		} else if len(expr.Args) > 1 {
			elemType := slice.Type.Elem()
			for _, el := range expr.Args[1:] {
				t := tc.checkExpr(el)
				if err := tc.isAssignableTo(t, el, elemType); err != nil {
					switch err.(type) {
					case invalidTypeInAssignment:
						panic(tc.errorf(expr, "%s in append", err))
					case nilConversionError:
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
			if _, ok := err.(nilConversionError); ok {
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
			tc.compilation.typeInfos[expr.Args[0]] = ti
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

// expandSpecialCall expands the special call f(g()) into f(t1, t2, ..., tn),
// where arg is the expression g(). If it is a special call, returns
// [t1, t2, ..., tn] and true, otherwise returns nil and false.
func (tc *typechecker) expandSpecialCall(arg ast.Expression) ([]ast.Expression, bool) {
	c, ok := arg.(*ast.Call)
	if !ok {
		return nil, false
	}
	tis := tc.checkCallExpression(c)
	switch len(tis) {
	case 0:
		panic(tc.errorf(arg, "%s used as value", arg))
	case 1:
		return nil, false
	default:
		args := make([]ast.Expression, len(tis))
		for i, ti := range tis {
			v := ast.NewCall(c.Pos(), c.Func, c.Args, false)
			tc.compilation.typeInfos[v] = ti
			args[i] = v
		}
		return args, true
	}
}

// checkCallExpression type checks a call expression, including type
// conversions and built-in function calls. Returns a list of typeinfos
// obtained from the call.
func (tc *typechecker) checkCallExpression(expr *ast.Call) []*typeInfo {

	// Check a builtin function call.
	if ident, ok := expr.Func.(*ast.Identifier); ok {
		if ti, ok := tc.lookupScopes(ident.Name, false); ok && ti.IsBuiltinFunction() {
			tc.compilation.typeInfos[expr.Func] = ti
			return tc.checkBuiltinCall(expr)
		}
	}

	t := tc.checkExprOrType(expr.Func)

	switch t.MethodType {
	case methodValueConcrete:
		t.MethodType = methodCallConcrete
	case methodValueInterface:
		t.MethodType = methodCallInterface
	}

	if t.Nil() {
		panic(tc.errorf(expr, "use of untyped nil"))
	}

	if t.IsType() {
		ti := tc.checkExplicitConversion(expr)
		return []*typeInfo{ti}
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

	special := false
	if len(args) == 1 && !callIsVariadic && numIn > 0 {
		var a []ast.Expression
		if a, special = tc.expandSpecialCall(args[0]); special {
			args = a
		}
	}

	if len(args) != numIn && (!funcIsVariadic || callIsVariadic || len(args) < numIn-1) {
		have := "("
		for i, arg := range args {
			if i > 0 {
				have += ", "
			}
			c := tc.compilation.typeInfos[arg]
			if c == nil {
				c = tc.checkExpr(arg)
			}
			if callIsVariadic && i == len(args)-1 && (c.Nil() || c.Type.AssignableTo(t.Type.In(numIn-1))) {
				have += "..." + t.Type.In(numIn-1).Elem().String()
			} else if c.Nil() {
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
		if i < lastIn || callIsVariadic || !funcIsVariadic {
			in = t.Type.In(i)
		} else if i == lastIn {
			in = t.Type.In(i).Elem()
		}
		a := tc.checkExpr(arg)
		if err := tc.isAssignableTo(a, arg, in); err != nil {
			if _, ok := err.(invalidTypeInAssignment); ok {
				if special {
					err = fmt.Errorf("cannot use %s value as type %s", a, in)
				}
				panic(tc.errorf(expr, "%s in argument to %s", err, expr.Func))
			}
			if _, ok := err.(nilConversionError); ok {
				panic(tc.errorf(args[i], "cannot use %s as type %s in argument to %s", a, in, expr.Func))
			}
			panic(tc.errorf(expr, "%s", err))
		}
		if a.Nil() {
			tc.compilation.typeInfos[expr.Args[i]] = tc.nilOf(in)
		} else {
			a.setValue(in)
		}
	}

	numOut := t.Type.NumOut()
	resultTypes := make([]*typeInfo, numOut)
	for i := 0; i < numOut; i++ {
		resultTypes[i] = &typeInfo{Type: t.Type.Out(i)}
	}

	return resultTypes
}

// isFormatType reports whether t is a format type.
func (tc *typechecker) isFormatType(t reflect.Type) bool {
	for _, name := range formatTypeName {
		ti, ok := tc.universe[name]
		if ok && ti.t.IsFormatType() && t == ti.t.Type {
			return true
		}
	}
	return false
}

// isMarkdown reports whether t is the markdown format type.
func (tc *typechecker) isMarkdown(t reflect.Type) bool {
	markdown, ok := tc.universe["markdown"]
	return ok && t == markdown.t.Type
}

// isHTML reports whether t is the html format type.
func (tc *typechecker) isHTML(t reflect.Type) bool {
	html, ok := tc.universe["html"]
	return ok && t == html.t.Type
}

func (tc *typechecker) checkExplicitConversion(expr *ast.Call) *typeInfo {

	t := tc.compilation.typeInfos[expr.Func]

	if len(expr.Args) == 0 {
		panic(tc.errorf(expr, "missing argument to conversion to %s: %s", t, expr))
	}
	if len(expr.Args) > 1 {
		panic(tc.errorf(expr, "too many arguments to conversion to %s: %s", t, expr))
	}

	arg := tc.checkExpr(expr.Args[0])

	// Check the special conversion from markdown to html.
	if t.IsFormatType() && tc.isMarkdown(arg.Type) && tc.isHTML(t.Type) {
		ti := &typeInfo{Type: t.Type}
		if arg.IsConstant() {
			var b bytes.Buffer
			r1 := tc.opts.renderer.WithOut(&b)
			r2 := r1.WithConversion(runtime.Format(ast.FormatMarkdown), runtime.Format(ast.FormatHTML))
			_, _ = r2.Out().Write([]byte(arg.Constant.String()))
			_ = r2.Close()
			_ = r1.Close()
			ti.Constant = stringConst(b.String())
			ti.setValue(t.Type)
		} else {
			arg.setValue(arg.Type)
		}
		return ti
	}

	var c constant
	var err error

	switch {
	case t.Type != stringType && t.IsFormatType() && t.Type != arg.Type && !arg.IsUntypedConstant():
		err = errTypeConversion
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
				n, _ := arg.Constant.representedBy(runeType)
				if n == nil {
					c = stringConst(string(unicode.ReplacementChar))
				} else {
					c = stringConst(rune(n.int64()))
				}
			case k == reflect.Slice && argKind == reflect.String:
				// As a special case, a string constant can be explicitly converted
				// to a slice of runes or bytes.
				if elem := t.Type.Elem(); elem != uint8Type && elem != int32Type {
					err = errTypeConversion
				}
			case t.Type.Kind() == reflect.Interface:
				if !arg.Type.Implements(t.Type) {
					err = errTypeConversion
				}
			default:
				c, err = arg.Constant.representedBy(t.Type)
				if err == errNotRepresentable {
					err = errTypeConversion
				}
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
			panic(tc.errorf(expr, "cannot convert %s (type %s) to type %s", expr.Args[0], arg, t.Type))
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

// checkCompositeLiteral type checks a composite literal.
//
// typ is the type of the composite literal, and it is taken in account only
// when node does not have a type. For example, given the following expression:
//
//     []T{{}, {}, {}}
//
// when checking the elements of the slice the typ argument passed to
// checkCompositeLiteral is 'T', and it is considered because the elements do
// not have an explicit type. In this other situation:
//
//     []T{T{}, T{}, T{}}
//
// every element specifies the type, so the argument 'typ' is simply ignored.
//
// The only case that cannot be handled by checkCompositeLiteral is when a
// composite literal element has an implicit type of kind pointer; in such case
// a tree transformation must be done externally.
func (tc *typechecker) checkCompositeLiteral(node *ast.CompositeLiteral, typ reflect.Type) *typeInfo {

	// Handle composite literal nodes with implicit type.
	if node.Type == nil {
		node.Type = ast.NewPlaceholder()
		tc.compilation.typeInfos[node.Type] = &typeInfo{Properties: propertyIsType, Type: typ}
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
	// tc.compilation.typeInfos[node.Type] = ti

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
				fieldTi, newName, err := tc.fieldByName(ti, ident.Name)
				if err != nil {
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
					tc.compilation.typeInfos[keyValue.Value] = valueTi
				} else {
					valueTi.setValue(fieldTi.Type)
				}
				ident.Name = newName
			}
		case false: // struct with implicit fields.
			if len(node.KeyValues) == 0 {
				ti := &typeInfo{Type: ti.Type}
				tc.compilation.typeInfos[node] = ti
				return ti
			}
			if len(node.KeyValues) < ti.Type.NumField() {
				panic(tc.errorf(node, "too few values in %s{...}", ti))
			}
			if len(node.KeyValues) > ti.Type.NumField() {
				panic(tc.errorf(node, "too many values in %s{...}", ti))
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
					tc.compilation.typeInfos[keyValue.Value] = valueTi
				} else {
					valueTi.setValue(fieldTi.Type)
				}
			}
		}

	case reflect.Slice, reflect.Array:

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
						panic(tc.errorf(node, "duplicate index in %s literal: %s", ti.Type.Kind(), kv.Key))
					}
					hasIndex[index] = struct{}{}
				}
			}
			var elemTi *typeInfo
			if cl, ok := kv.Value.(*ast.CompositeLiteral); ok {
				if ti.Type.Elem().Kind() == reflect.Ptr {
					// The slice (or array) as element *T, so the value '{..}' must be
					// replaced with '&T{..}'.
					kv.Value = ast.NewUnaryOperator(cl.Pos(), ast.OperatorAddress, cl)
					tc.checkCompositeLiteral(cl, ti.Type.Elem().Elem()) // []*T -> T (or [n]*T -> T)
					elemTi = tc.checkExpr(kv.Value)
				} else {
					elemTi = tc.checkCompositeLiteral(cl, ti.Type.Elem())
				}
			} else {
				elemTi = tc.checkExpr(kv.Value)
			}
			if err := tc.isAssignableTo(elemTi, kv.Value, ti.Type.Elem()); err != nil {
				if _, ok := err.(invalidTypeInAssignment); ok {
					panic(tc.errorf(node, "%s in %s literal", err, ti.Type.Kind()))
				}
				panic(tc.errorf(node, "%s", err))
			}
			if elemTi.Nil() {
				elemTi = tc.nilOf(ti.Type.Elem())
				tc.compilation.typeInfos[kv.Value] = elemTi
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
				if keyType.Kind() == reflect.Ptr {
					// The map as the key type *T, so the value '{..}' must be
					// replaced with '&T{..}'.
					kv.Key = ast.NewUnaryOperator(compLit.Pos(), ast.OperatorAddress, compLit)
					tc.checkCompositeLiteral(compLit, keyType.Elem()) // *T -> T
					keyTi = tc.checkExpr(kv.Key)
				} else {
					keyTi = tc.checkCompositeLiteral(compLit, keyType)
				}
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
				tc.compilation.typeInfos[kv.Key] = keyTi
			} else {
				keyTi.setValue(keyType)
			}
			var valueTi *typeInfo
			if cl, ok := kv.Value.(*ast.CompositeLiteral); ok {
				if elemType.Kind() == reflect.Ptr {
					// The map as the element type *T, so the value '{..}' must
					// be replaced with '&T{..}'.
					kv.Value = ast.NewUnaryOperator(cl.Pos(), ast.OperatorAddress, cl)
					tc.checkCompositeLiteral(cl, elemType.Elem()) // *T -> T
					valueTi = tc.checkExpr(kv.Value)
				} else {
					valueTi = tc.checkCompositeLiteral(cl, elemType)
				}
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
				tc.compilation.typeInfos[kv.Value] = valueTi
			} else {
				valueTi.setValue(elemType)
			}
		}

	default:

		panic(tc.errorf(node, "invalid type for composite literal: %s", ti))

	}

	nodeTi := &typeInfo{Type: ti.Type}
	tc.compilation.typeInfos[node] = nodeTi

	return nodeTi
}

// builtinCallName returns the name of the builtin function in a call
// expression. If expr is not a call expression or the function is not a
// builtin, it returns an empty string.
func (tc *typechecker) builtinCallName(expr ast.Expression) string {
	if call, ok := expr.(*ast.Call); ok && tc.compilation.typeInfos[call.Func].IsBuiltinFunction() {
		return call.Func.(*ast.Identifier).Name
	}
	return ""
}

// isCompileConstant reports whether expr does not contain channel receives or
// non-constant function calls.
func (tc *typechecker) isCompileConstant(expr ast.Expression) bool {
	if ti := tc.compilation.typeInfos[expr]; ti.IsConstant() {
		return true
	}
	switch expr := expr.(type) {
	case *ast.BinaryOperator:
		return tc.isCompileConstant(expr.Expr1) && tc.isCompileConstant(expr.Expr2)
	case *ast.Call:
		ti := tc.compilation.typeInfos[expr.Func]
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

// checkDollarIdentifier type checks a dollar identifier $x.
func (tc *typechecker) checkDollarIdentifier(expr *ast.DollarIdentifier) *typeInfo {

	// Check that x is a valid identifier.
	if ti, ok := tc.lookupScopes(expr.Ident.Name, false); ok {
		// Check that x is not a builtin function.
		if ti.IsBuiltinFunction() {
			panic(tc.errorf(expr.Ident, "use of builtin %s not in function call", expr.Ident))
		}
		// Check that x is not a type.
		if ti.IsType() {
			panic(tc.errorf(expr.Ident, "unexpected type in dollar identifier"))
		}
		// Check that x is not a local identifier.
		if tc.isLocallyDeclared(expr.Ident.Name) {
			panic(tc.errorf(expr, "use of local identifier within dollar identifier"))
		}
		// Check that x is not declared in the file/package block, that
		// contains, for example, the variable declarations at the top level of
		// an imported or extending file.
		if tc.isDeclaredInFilePackageBlock(expr.Ident.Name) {
			panic(tc.errorf(expr, "use of top-level identifier within dollar identifier"))
		}
	}

	// Set the IR of the expression.
	var arg *ast.Identifier
	var pos = expr.Pos()
	if _, isGlobal := tc.globalScope[expr.Ident.Name]; isGlobal {
		arg = expr.Ident // "x"
	} else {
		arg = ast.NewIdentifier(pos, "nil") // "nil"
	}
	// expr.IR.Ident is set to "interface{}(x)" or "interface{}(nil)".
	expr.IR.Ident = ast.NewCall(
		pos,
		ast.NewInterface(pos), // "interface{}"
		[]ast.Expression{arg}, // "x" or "nil"
		false,
	)

	// Type check the IR of the expression and return its type info.
	return tc.checkExpr(expr.IR.Ident)
}

// checkDefault type checks a default expression. show indicates if the
// default expression is in a show statement.
func (tc *typechecker) checkDefault(expr *ast.Default, show bool) typeInfoPair {

	var tis typeInfoPair

	switch n := expr.Expr1.(type) {

	case *ast.Identifier:
		if isBlankIdentifier(n) {
			panic(tc.errorf(n, "cannot use _ as value"))
		}
		var ok bool
		if tis[0], ok = tc.lookupScopes(n.Name, false); ok {
			if tis[0].IsPackage() {
				panic(tc.errorf(expr, "use of package %s without selector", n))
			}
			if tis[0].Nil() {
				panic(tc.errorf(n, "use of untyped nil"))
			}
			if tis[0].InUniverse() && n.Name == "this" || strings.HasPrefix(n.Name, "$this") {
				panic(tc.errorf(n, "use of predeclared identifier this"))
			}
			if tis[0].IsBuiltinFunction() {
				panic(tc.errorf(n, "use of builtin %s not in function call", n))
			}
			if !tis[0].InUniverse() && !tis[0].Global() {
				panic(tc.errorf(n, "use of non-builtin on left side of default"))
			}
			if tis[0].IsType() {
				panic(tc.errorf(n, "unexpected type on left side of default"))
			}
			if tis[0].InUniverse() && n.Name == "iota" {
				tis[0] = nil
				if tc.iota >= 0 {
					tis[0] = &typeInfo{
						Constant:   int64Const(tc.iota),
						Type:       intType,
						Properties: propertyUntyped,
					}
				}
			}
		}

	case *ast.Call:
		if !tc.inExtendedFile {
			panic(tc.errorf(n, "use of default with call in non-extended file"))
		}
		ident := n.Func.(*ast.Identifier)
		if isBlankIdentifier(ident) {
			panic(tc.errorf(ident, "cannot use _ as value"))
		}
		if ti, ok := tc.lookupScopes(ident.Name, false); ok {
			// TODO(Gianluca): test 'nil() default'
			if ti.InUniverse() && ident.Name == "this" || strings.HasPrefix(ident.Name, "$this") {
				panic(tc.errorf(n, "use of predeclared identifier this"))
			}
			if ti.IsType() {
				panic(tc.errorf(ident, "type conversion on left side of default"))
			}
			if ti.IsBuiltinFunction() {
				panic(tc.errorf(ident, "use of builtin %s on left side of default", ident))
			}
			if !ti.IsMacroDeclaration() {
				panic(tc.errorf(ident, "cannot use %s (type %s) as macro", ident, ti.Type))
			}
			if !ti.MacroDeclaredInExtendingFile() {
				panic(tc.errorf(ident, "macro not declared in file with extends"))
			}
			tis[0] = tc.checkExpr(n)
		} else {
			// Call arguments are type checked but never executed.
			for i, arg := range n.Args {
				ti := tc.checkExpr(arg)
				tc.mustBeAssignableTo(ti, arg, emptyInterfaceType, false, nil)
				// Check variadic calls.
				if n.IsVariadic && i == len(n.Args)-1 && ti.Type.Kind() != reflect.Slice {
					panic(tc.errorf(arg, "cannot use %s (type %s) as variadic argument", arg, ti.Type))
				}
			}
		}

	case *ast.Render:
		if n.Tree != nil {
			tis[0] = tc.checkRender(n)
		}

	default:
		panic("unexpected default")
	}

	toBeEmitted := tc.toBeEmitted
	if tis[0] != nil {
		tc.compilation.typeInfos[expr.Expr1] = tis[0]
		tc.toBeEmitted = false
	}
	tis[1] = tc.checkExpr(expr.Expr2)
	tc.toBeEmitted = toBeEmitted

	if _, ok := expr.Expr1.(*ast.Identifier); !ok && !show {
		if typ := tis[1].Type; !tc.isFormatType(typ) {
			panic(tc.errorf(expr.Expr2, "mismatched format type and %s type", typ))
		}
	}

	tc.compilation.typeInfos[expr] = tis.TypeInfo()

	return tis
}

// checkRender checks the 'render' expression and changes its internal
// representation.
func (tc *typechecker) checkRender(render *ast.Render) *typeInfo {

	// Transform the render expression to a call to a macro that has the
	// rendered file as body.

	tree := render.Tree

	stored, ok := tc.compilation.renderImportMacro[tree]
	if !ok {
		macroDecl := ast.NewFunc(
			nil,
			ast.NewIdentifier(nil, strconv.Quote(tree.Path)),
			ast.NewFuncType(nil, true, nil, nil, false), // func()
			ast.NewBlock(nil, tree.Nodes),
			false,
			tree.Format,
		)
		// The same 'import' statement may be shared by different template
		// files that 'render' the same file. This is the expected and intended
		// behavior.
		importt := ast.NewImport(nil, nil, "/"+render.Path)
		importt.Tree = tree
		importt.Tree.Nodes = []ast.Node{macroDecl}
		stored.Macro = macroDecl
		stored.Import = importt
		tc.compilation.renderImportMacro[tree] = stored
	}

	render.IR.Call = ast.NewCall(render.Pos(), stored.Macro.Ident, nil, false)
	render.IR.Import = stored.Import

	// The same 'import' statement may be type checked more than once per file.
	// This is the expected and intended behavior.
	tc.checkNodes([]ast.Node{stored.Import})

	return tc.checkExpr(render.IR.Call)
}
