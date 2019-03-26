// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"unicode"

	"scrigo/ast"
)

var errDivisionByZero = errors.New("division by zero")

const noEllipses = -1

type scopeElement struct {
	t    *TypeInfo
	decl *ast.Identifier
}

type typeCheckerScope map[string]scopeElement

type HTML string

var boolType = reflect.TypeOf(false)
var stringType = reflect.TypeOf("")
var intType = reflect.TypeOf(0)
var uint8Type = reflect.TypeOf(uint8(0))
var int32Type = reflect.TypeOf(int32(0))
var float64Type = reflect.TypeOf(float64(0))
var emptyInterfaceType = reflect.TypeOf(&[]interface{}{interface{}(nil)}[0]).Elem()

var builtinTypeInfo = &TypeInfo{Properties: PropertyIsBuiltin}
var uint8TypeInfo = &TypeInfo{Type: uint8Type, Properties: PropertyIsType}
var int32TypeInfo = &TypeInfo{Type: int32Type, Properties: PropertyIsType}

var untypedBoolTypeInfo = &TypeInfo{Type: boolType, Properties: PropertyUntyped}

var universe = typeCheckerScope{
	"append":      {t: builtinTypeInfo},
	"cap":         {t: builtinTypeInfo},
	"close":       {t: builtinTypeInfo},
	"complex":     {t: builtinTypeInfo},
	"copy":        {t: builtinTypeInfo},
	"delete":      {t: builtinTypeInfo},
	"imag":        {t: builtinTypeInfo},
	"len":         {t: builtinTypeInfo},
	"make":        {t: builtinTypeInfo},
	"new":         {t: builtinTypeInfo},
	"nil":         {t: &TypeInfo{Properties: PropertyNil}},
	"panic":       {t: builtinTypeInfo},
	"print":       {t: builtinTypeInfo},
	"println":     {t: builtinTypeInfo},
	"real":        {t: builtinTypeInfo},
	"recover":     {t: builtinTypeInfo},
	"byte":        {t: uint8TypeInfo},
	"bool":        {t: &TypeInfo{Type: boolType, Properties: PropertyIsType}},
	"complex128":  {t: &TypeInfo{Type: reflect.TypeOf(complex128(0)), Properties: PropertyIsType}},
	"complex64":   {t: &TypeInfo{Type: reflect.TypeOf(complex64(0)), Properties: PropertyIsType}},
	"error":       {t: &TypeInfo{Type: reflect.TypeOf((*error)(nil)), Properties: PropertyIsType}},
	"float32":     {t: &TypeInfo{Type: reflect.TypeOf(float32(0)), Properties: PropertyIsType}},
	"float64":     {t: &TypeInfo{Type: float64Type, Properties: PropertyIsType}},
	"false":       {t: &TypeInfo{Type: boolType, Properties: PropertyIsConstant | PropertyUntyped, Value: false}},
	"int":         {t: &TypeInfo{Type: intType, Properties: PropertyIsType}},
	"int16":       {t: &TypeInfo{Type: reflect.TypeOf(int16(0)), Properties: PropertyIsType}},
	"int32":       {t: int32TypeInfo},
	"int64":       {t: &TypeInfo{Type: reflect.TypeOf(int64(0)), Properties: PropertyIsType}},
	"int8":        {t: &TypeInfo{Type: reflect.TypeOf(int8(0)), Properties: PropertyIsType}},
	"interface{}": {t: &TypeInfo{Type: emptyInterfaceType, Properties: PropertyIsType}},
	"rune":        {t: int32TypeInfo},
	"string":      {t: &TypeInfo{Type: stringType, Properties: PropertyIsType}},
	"true":        {t: &TypeInfo{Type: boolType, Properties: PropertyIsConstant | PropertyUntyped, Value: true}},
	"uint":        {t: &TypeInfo{Type: reflect.TypeOf(uint(0)), Properties: PropertyIsType}},
	"uint16":      {t: &TypeInfo{Type: reflect.TypeOf(uint32(0)), Properties: PropertyIsType}},
	"uint32":      {t: &TypeInfo{Type: reflect.TypeOf(uint32(0)), Properties: PropertyIsType}},
	"uint64":      {t: &TypeInfo{Type: reflect.TypeOf(uint64(0)), Properties: PropertyIsType}},
	"uint8":       {t: uint8TypeInfo},
	"uintptr":     {t: &TypeInfo{Type: reflect.TypeOf(uintptr(0)), Properties: PropertyIsType}},
}

type ancestor struct {
	scopeLevel int
	node       ast.Node
}

type DeclarationType int

const (
	DeclarationConstant = iota + 1
	DeclarationVariable
	DeclarationFunction
)

// Declaration is a package global declaration.
type Declaration struct {
	Node            ast.Node        // ast node of the declaration.
	Ident           string          // identifier of the declaration.
	Type            ast.Expression  // nil if declaration has no type.
	DeclarationType DeclarationType // constant, variable or function.
	Value           ast.Node        // ast.Expression for variables/constant, ast.Block for functions.
}

type scopeVariable struct {
	ident      string
	scopeLevel int
	node       ast.Node
}

// typechecker represents the state of a type checking.
type typechecker struct {
	path             string
	imports          map[string]PackageInfo // TODO (Gianluca): review!
	universe         typeCheckerScope
	filePackageBlock typeCheckerScope
	scopes           []typeCheckerScope
	ancestors        []*ancestor
	terminating      bool // https://golang.org/ref/spec#Terminating_statements
	hasBreak         map[ast.Node]bool
	unusedVars       []*scopeVariable
	unusedImports    map[string][]string
	typeInfo         map[ast.Node]*TypeInfo
	upValues         map[*ast.Identifier]bool

	// Variable initialization support structures.
	// TODO (Gianluca): can be simplified?
	declarations        []*Declaration      // global declarations.
	initOrder           []string            // global variables initialization order.
	varDeps             map[string][]string // key is a variable, value is list of its dependencies.
	currentIdent        string              // identifier currently being evaluated.
	currentlyEvaluating []string            // stack of identifiers used in a single evaluation.
	temporaryEvaluated  map[string]*TypeInfo
}

func newTypechecker() *typechecker {
	return &typechecker{
		imports:            make(map[string]PackageInfo),
		universe:           make(typeCheckerScope),
		filePackageBlock:   make(typeCheckerScope),
		hasBreak:           make(map[ast.Node]bool),
		unusedImports:      make(map[string][]string),
		typeInfo:           make(map[ast.Node]*TypeInfo),
		upValues:           make(map[*ast.Identifier]bool),
		varDeps:            make(map[string][]string),
		temporaryEvaluated: make(map[string]*TypeInfo),
	}
}

// getDecl returns the declaration called name, or nil if it does not exist.
func (tc *typechecker) getDecl(name string) *Declaration {
	for _, v := range tc.declarations {
		if name == v.Ident {
			return v
		}
	}
	return nil
}

// addScope adds a new empty scope to the type checker.
func (tc *typechecker) addScope() {
	tc.scopes = append(tc.scopes, make(typeCheckerScope))
}

// removeCurrentScope removes the current scope from the type checker.
func (tc *typechecker) removeCurrentScope() {
	cut := len(tc.unusedVars)
	for i := len(tc.unusedVars) - 1; i >= 0; i-- {
		v := tc.unusedVars[i]
		if v.scopeLevel < len(tc.scopes)-1 {
			break
		}
		if v.node != nil {
			panic(tc.errorf(v.node, "%s declared and not used", v.ident))
		}
		cut = i
	}
	tc.unusedVars = tc.unusedVars[:cut]
	tc.scopes = tc.scopes[:len(tc.scopes)-1]
}

// lookupScopes looks up name in the scopes. Returns the type info of the name or
// false if the name does not exist. If justCurrentScope is true, lookupScopes
// looks up only in the current scope.
func (tc *typechecker) lookupScopes(name string, justCurrentScope bool) (*TypeInfo, bool) {
	// Iterating over scopes, from inside.
	for i := len(tc.scopes) - 1; i >= 0; i-- {
		elem, ok := tc.scopes[i][name]
		if ok {
			return elem.t, true
		}
		if justCurrentScope && i == len(tc.scopes)-1 {
			return nil, false
		}
	}
	// Package + file block.
	if elem, ok := tc.filePackageBlock[name]; ok {
		return elem.t, true
	}
	// Universe.
	if elem, ok := tc.universe[name]; ok {
		return elem.t, true
	}
	return nil, false
}

// assignScope assigns value to name in the last scope.
func (tc *typechecker) assignScope(name string, value *TypeInfo, declNode *ast.Identifier) {
	tc.scopes[len(tc.scopes)-1][name] = scopeElement{t: value, decl: declNode}
}

func (tc *typechecker) addToAncestors(n ast.Node) {
	tc.ancestors = append(tc.ancestors, &ancestor{len(tc.scopes), n})
}

func (tc *typechecker) removeLastAncestor() {
	tc.ancestors = tc.ancestors[:len(tc.ancestors)-1]
}

// getCurrentFunc returns the current function and the related scope level. If
// getCurrentFunc is called when not in a function body, returns nil and 0.
func (tc *typechecker) getCurrentFunc() (*ast.Func, int) {
	for i := len(tc.ancestors) - 1; i >= 0; i-- {
		if f, ok := tc.ancestors[i].node.(*ast.Func); ok {
			return f, tc.ancestors[i].scopeLevel
		}
	}
	return nil, 0
}

// isUpValue checks if name is an upvalue.
func (tc *typechecker) isUpValue(name string) bool {
	_, funcBound := tc.getCurrentFunc()
	for i := len(tc.scopes) - 1; i >= 0; i-- {
		for n := range tc.scopes[i] {
			if n != name {
				continue
			}
			if i < funcBound-1 { // out of current function scope.
				tc.upValues[tc.scopes[i][n].decl] = true
				return true
			}
			return false
		}
	}
	return false
}

func (tc *typechecker) checkIdentifier(ident *ast.Identifier, using bool) *TypeInfo {

	// Upvalues.
	if fun, _ := tc.getCurrentFunc(); fun != nil {
		if tc.isUpValue(ident.Name) {
			fun.Upvalues = append(fun.Upvalues, ident.Name)
		}
	}

	i, ok := tc.lookupScopes(ident.Name, false)
	if !ok {
		panic(tc.errorf(ident, "undefined: %s", ident.Name))
	}

	if i == builtinTypeInfo {
		panic(tc.errorf(ident, "use of builtin %s not in function call", ident.Name))
	}

	// For "." imported packages, marks package as used.
ImportsLoop:
	for pkg, decls := range tc.unusedImports {
		for _, d := range decls {
			if d != ident.Name {
				delete(tc.unusedImports, pkg)
				break ImportsLoop
			}
		}
	}

	if tmpTi, ok := tc.temporaryEvaluated[ident.Name]; ok {
		return tmpTi
	}

	if tc.getDecl(ident.Name) != nil {
		tc.varDeps[tc.currentIdent] = append(tc.varDeps[tc.currentIdent], ident.Name)
		tc.currentlyEvaluating = append(tc.currentlyEvaluating, ident.Name)
		if containsDuplicates(tc.currentlyEvaluating) {
			if d := tc.getDecl(tc.currentIdent); d != nil && d.DeclarationType == DeclarationFunction {
				ti, _ := tc.lookupScopes(ident.Name, false)
				return ti
			}
			// TODO (Gianluca): add positions.
			panic(tc.errorf(ident, "initialization loop:\n\t%s", strings.Join(tc.currentlyEvaluating, " refers to\n\t")))
		}
	}

	// Check bodies of global functions.
	// TODO (Gianluca): this must be done only when checking global variables.
	if i.Type != nil && i.Type.Kind() == reflect.Func && !i.Addressable() {
		decl := tc.getDecl(ident.Name)
		tc.addScope()
		tc.ancestors = append(tc.ancestors, &ancestor{len(tc.scopes), decl.Node})
		// Adds parameters to the function body scope.
		params := fillParametersTypes(decl.Node.(*ast.Func).Type.Parameters)
		isVariadic := decl.Node.(*ast.Func).Type.IsVariadic
		for i, param := range params {
			if param.Ident != nil {
				t := tc.checkType(param.Type, noEllipses)
				if isVariadic && i == len(params)-1 {
					tc.assignScope(param.Ident.Name, &TypeInfo{Type: reflect.SliceOf(t.Type), Properties: PropertyAddressable}, nil)
				} else {
					tc.assignScope(param.Ident.Name, &TypeInfo{Type: t.Type, Properties: PropertyAddressable}, nil)
				}
			}
		}
		// Adds named return values to the function body scope.
		for _, ret := range fillParametersTypes(decl.Node.(*ast.Func).Type.Result) {
			t := tc.checkType(ret.Type, noEllipses)
			if ret.Ident != nil {
				tc.assignScope(ret.Ident.Name, &TypeInfo{Type: t.Type, Properties: PropertyAddressable}, nil)
			}
		}
		tc.checkNodes(decl.Value.(*ast.Block).Nodes)
		tc.ancestors = tc.ancestors[:len(tc.ancestors)-1]
		tc.removeCurrentScope()
	}

	// Global declaration.
	if i == notChecked {
		switch d := tc.getDecl(ident.Name); d.DeclarationType {
		case DeclarationConstant:
			ti := tc.checkExpression(d.Value.(ast.Expression))
			tc.temporaryEvaluated[ident.Name] = ti
			return ti
		case DeclarationVariable:
			ti := tc.checkExpression(d.Value.(ast.Expression))
			ti.Properties |= PropertyAddressable
			tc.temporaryEvaluated[ident.Name] = ti
			return ti
		case DeclarationFunction:
			tc.checkNodes(d.Value.(*ast.Block).Nodes)
			return &TypeInfo{Type: tc.typeof(d.Type, noEllipses).Type}
		}
	}

	if using {
		for i := len(tc.unusedVars) - 1; i >= 0; i-- {
			v := tc.unusedVars[i]
			if v.ident == ident.Name {
				v.node = nil
				break
			}
		}
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
func (tc *typechecker) checkExpression(expr ast.Expression) *TypeInfo {
	if isBlankIdentifier(expr) {
		panic(tc.errorf(expr, "cannot use _ as value"))
	}
	ti := tc.typeof(expr, noEllipses)
	if ti.IsType() {
		panic(tc.errorf(expr, "type %s is not an expression", ti))
	}
	tc.typeInfo[expr] = ti
	return ti
}

// checkType evaluates expr as a type and returns the type info. Returns an
// error if expr is not an type.
func (tc *typechecker) checkType(expr ast.Expression, length int) *TypeInfo {
	if isBlankIdentifier(expr) {
		panic(tc.errorf(expr, "cannot use _ as value"))
	}
	if ptr, ok := expr.(*ast.UnaryOperator); ok && ptr.Operator() == ast.OperatorMultiplication {
		ti := tc.typeof(ptr.Expr, length)
		if !ti.IsType() {
			panic(tc.errorf(expr, "%s is not a type", expr))
		}
		newTi := &TypeInfo{Properties: PropertyIsType, Type: reflect.PtrTo(ti.Type)}
		tc.typeInfo[expr] = newTi
		return newTi
	}
	ti := tc.typeof(expr, length)
	if !ti.IsType() {
		panic(tc.errorf(expr, "%s is not a type", expr))
	}
	tc.typeInfo[expr] = ti
	return ti
}

// typeof returns the type of expr. If expr is not an expression but a type,
// returns the type.
func (tc *typechecker) typeof(expr ast.Expression, length int) *TypeInfo {

	switch expr := expr.(type) {

	case *ast.String:
		return &TypeInfo{
			Type:       stringType,
			Properties: PropertyUntyped | PropertyIsConstant,
			Value:      expr.Text,
		}

	case *ast.Int:
		return &TypeInfo{
			Type:       intType,
			Properties: PropertyUntyped | PropertyIsConstant,
			Value:      &expr.Value,
		}

	case *ast.Rune:
		return &TypeInfo{
			Type:       int32Type,
			Properties: PropertyUntyped | PropertyIsConstant,
			Value:      big.NewInt(int64(expr.Value)),
		}

	case *ast.Float:
		return &TypeInfo{
			Type:       float64Type,
			Properties: PropertyUntyped | PropertyIsConstant,
			Value:      &expr.Value,
		}

	case *ast.Parenthesis:
		panic("unexpected parenthesis")

	case *ast.UnaryOperator:
		_ = tc.checkExpression(expr.Expr)
		t, err := unaryOp(tc.typeInfo[expr.Expr], expr)
		if err != nil {
			panic(tc.errorf(expr, "%s", err))
		}
		return t

	case *ast.BinaryOperator:
		t, err := tc.binaryOp(expr)
		if err != nil {
			panic(tc.errorf(expr, "%s", err))
		}
		return t

	case *ast.Identifier:
		t := tc.checkIdentifier(expr, true)
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
		return &TypeInfo{Properties: PropertyIsType, Type: reflect.MapOf(key.Type, value.Type)}

	case *ast.SliceType:
		elem := tc.checkType(expr.ElementType, noEllipses)
		return &TypeInfo{Properties: PropertyIsType, Type: reflect.SliceOf(elem.Type)}

	case *ast.ArrayType:
		elem := tc.checkType(expr.ElementType, noEllipses)
		if expr.Len == nil { // ellipsis.
			return &TypeInfo{Properties: PropertyIsType, Type: reflect.ArrayOf(length, elem.Type)}
		}
		len := tc.checkExpression(expr.Len)
		if !len.IsConstant() {
			panic(tc.errorf(expr, "non-constant array bound %s", expr.Len))
		}
		declLength, err := convertImplicit(len, intType)
		if err != nil {
			panic(tc.errorf(expr, err.Error()))
		}
		n := int(declLength.(*big.Int).Int64())
		if n < 0 {
			panic(tc.errorf(expr, "array bound must be non-negative"))
		}
		if length > n {
			panic(tc.errorf(expr, "array index %d out of bounds [0:%d]", length-1, n))
		}
		return &TypeInfo{Properties: PropertyIsType, Type: reflect.ArrayOf(n, elem.Type)}

	case *ast.CompositeLiteral:
		return tc.checkCompositeLiteral(expr, nil)

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
			if res.Type == nil {
				out[i] = out[i+1]
			} else {
				c := tc.checkType(res.Type, noEllipses)
				out[i] = c.Type
			}
		}
		return &TypeInfo{Type: reflect.FuncOf(in, out, variadic), Properties: PropertyIsType}

	case *ast.Func:
		tc.addScope()
		t := tc.checkType(expr.Type, noEllipses)
		tc.ancestors = append(tc.ancestors, &ancestor{len(tc.scopes), expr})
		// Adds parameters to the function body scope.
		params := fillParametersTypes(expr.Type.Parameters)
		isVariadic := expr.Type.IsVariadic
		for i, f := range params {
			if f.Ident != nil {
				t := tc.checkType(f.Type, noEllipses)
				if isVariadic && i == len(params)-1 {
					tc.assignScope(f.Ident.Name, &TypeInfo{Type: reflect.SliceOf(t.Type), Properties: PropertyAddressable}, nil)
					continue
				}
				tc.assignScope(f.Ident.Name, &TypeInfo{Type: t.Type, Properties: PropertyAddressable}, nil)
			}
		}
		// Adds named return values to the function body scope.
		for _, f := range fillParametersTypes(expr.Type.Result) {
			if f.Ident != nil {
				t := tc.checkType(f.Type, noEllipses)
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
		tc.removeCurrentScope()
		return &TypeInfo{Type: t.Type}

	case *ast.Call:
		types, _ := tc.checkCallExpression(expr, false)
		if len(types) == 0 {
			panic(tc.errorf(expr, "%v used as value", expr))
		}
		if len(types) > 1 {
			panic(tc.errorf(expr, "multiple-value %v in single-value context", expr))
		}
		return types[0]

	case *ast.Index:
		t := tc.checkExpression(expr.Expr)
		if t.Nil() {
			panic(tc.errorf(expr, "use of untyped nil"))
		}
		kind := t.Type.Kind()
		switch kind {
		case reflect.Slice, reflect.String, reflect.Array, reflect.Ptr:
			realType := t.Type
			realKind := t.Type.Kind()
			if kind == reflect.Ptr {
				realType = t.Type.Elem()
				realKind = realType.Kind()
				if realKind != reflect.Array {
					panic(tc.errorf(expr, "invalid operation: %v (type %s does not support indexing)", expr, t))
				}
			}
			_ = tc.checkIndex(expr.Index, t, false)
			var typ reflect.Type
			switch kind {
			case reflect.String:
				typ = universe["byte"].t.Type
			case reflect.Slice, reflect.Array:
				typ = t.Type.Elem()
			case reflect.Ptr:
				typ = t.Type.Elem().Elem()
			}
			ti := &TypeInfo{Type: typ}
			if kind == reflect.Slice || kind == reflect.Array && t.Addressable() || kind == reflect.Ptr {
				ti.Properties = PropertyAddressable
			}
			return ti
		case reflect.Map:
			key := tc.checkExpression(expr.Index)
			if !isAssignableTo(key, t.Type.Key()) {
				if key.Nil() {
					panic(tc.errorf(expr, "cannot convert nil to type %s", t.Type.Key()))
				}
				panic(tc.errorf(expr, "cannot use %s (type %s) as type %s in map index", expr.Index, key.ShortString(), t.Type.Key()))
			}
			return &TypeInfo{Type: t.Type.Elem()}
		default:
			panic(tc.errorf(expr, "invalid operation: %s (type %s does not support indexing)", expr, t.ShortString()))
		}

	case *ast.Slicing:
		// TODO(marco) support full slice expressions
		t := tc.checkExpression(expr.Expr)
		if t.Nil() {
			panic(tc.errorf(expr, "use of untyped nil"))
		}
		kind := t.Type.Kind()
		realType := t.Type
		realKind := kind
		switch kind {
		case reflect.String, reflect.Slice:
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
		lv, hv := -1, -1
		if expr.Low != nil {
			lv = tc.checkIndex(expr.Low, t, true)
		}
		if expr.High != nil {
			hv = tc.checkIndex(expr.High, t, true)
		}
		if lv != -1 && hv != -1 && lv > hv {
			panic(tc.errorf(expr, "invalid slice index: %d > %d", lv, hv))
		}
		switch kind {
		case reflect.String, reflect.Slice:
			return &TypeInfo{Type: t.Type}
		case reflect.Array, reflect.Ptr:
			return &TypeInfo{Type: reflect.SliceOf(realType.Elem())}
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
					pkg := ti.Value.(*PackageInfo)
					v, ok := pkg.Declarations[expr.Ident]
					if !ok {
						panic(tc.errorf(expr, "undefined: %v", expr))
					}
					return v
				}
			}
		}
		t := tc.typeof(expr.Expr, noEllipses)
		if t.IsType() {
			method, ok := methodByName(t, expr.Ident)
			if !ok {
				panic(tc.errorf(expr, "%v undefined (type %s has no method %s)", expr, t, expr.Ident))
			}
			return method
		}
		if t.Type.Kind() == reflect.Ptr {
			method, ok := methodByName(t, expr.Ident)
			if ok {
				return method
			}
			field, ok := fieldByName(t, expr.Ident)
			if ok {
				return field
			}
			panic(tc.errorf(expr, "%v undefined (type %s has no field or method %s)", expr, t, expr.Ident))
		}
		method, ok := methodByName(t, expr.Ident)
		if ok {
			return method
		}
		field, ok := fieldByName(t, expr.Ident)
		if ok {
			return field
		}
		panic(tc.errorf(expr, "%v undefined (type %s has no field or method %s)", expr, t, expr.Ident))

	case *ast.TypeAssertion:
		t := tc.typeof(expr.Expr, noEllipses)
		if t.Type.Kind() != reflect.Interface {
			panic(tc.errorf(expr, "invalid type assertion: %v (non-interface type %s on left)", expr, t))
		}
		tc.typeInfo[expr.Expr] = t
		t = tc.checkType(expr.Type, noEllipses)
		tc.typeInfo[expr.Type] = t
		return t

	}

	panic(fmt.Errorf("unexpected: %v (type %T)", expr, expr))
}

// checkIndex checks the type of expr as an index in a index or slice
// expression. If it is a constant returns the integer value, otherwise
// returns -1.
func (tc *typechecker) checkIndex(expr ast.Expression, t *TypeInfo, isSlice bool) int {
	typ := t.Type
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	index := tc.checkExpression(expr)
	if index.Untyped() && !index.IsNumeric() || !index.Untyped() && !index.IsInteger() {
		if isSlice {
			panic(tc.errorf(expr, "invalid slice index %s (type %s)", expr, index))
		}
		panic(tc.errorf(expr, "non-integer %s index %s", typ.Kind(), expr))
	}
	i := -1
	if index.IsConstant() {
		n, err := representedBy(index, intType)
		if err != nil {
			panic(tc.errorf(expr, fmt.Sprintf("%s", err)))
		}
		i = int(n.(*big.Int).Int64())
		if i < 0 {
			panic(tc.errorf(expr, "invalid %s index %s (index must be non-negative)", typ.Kind(), expr))
		}
		j := i
		if isSlice {
			j--
		}
		if t.IsConstant() {
			if s := t.Value.(string); j >= len(s) {
				what := typ.Kind().String()
				if isSlice {
					what = "slice"
				}
				panic(tc.errorf(expr, "invalid %s index %s (out of bounds for %d-byte string)", what, expr, len(s)))
			}
		} else if typ.Kind() == reflect.Array && j > typ.Len() {
			panic(tc.errorf(expr, "invalid array index %s (out of bounds for %d-element array)", expr, typ.Len()))
		}
	}
	return i
}

// binaryOp executes the binary expression t1 op t2 and returns its result.
// Returns an error if the operation can not be executed.
func (tc *typechecker) binaryOp(expr *ast.BinaryOperator) (*TypeInfo, error) {

	t1 := tc.checkExpression(expr.Expr1)
	t2 := tc.checkExpression(expr.Expr2)

	if t1.Untyped() && t2.Untyped() {
		return uBinaryOp(t1, expr, t2)
	}

	op := expr.Op

	if t1.Nil() || t2.Nil() {
		if t1.Nil() && t2.Nil() {
			return nil, fmt.Errorf("invalid operation: %v (operator %s not defined on nil)", expr, op)
		}
		t := t1
		if t.Nil() {
			t = t2
		}
		k := t.Type.Kind()
		if !operatorsOfKind[k][op] {
			return nil, fmt.Errorf("invalid operation: %v (operator %s not defined on %s)", expr, op, k)
		}
		if !t.Type.Comparable() {
			return nil, fmt.Errorf("cannot convert nil to type %s", t)
		}
		if op != ast.OperatorEqual && op != ast.OperatorNotEqual {
			return nil, fmt.Errorf("invalid operation: %v (operator %s not defined on %s)", expr, op, k)
		}
		return untypedBoolTypeInfo, nil
	}

	if t1.Untyped() {
		v, err := convertImplicit(t1, t2.Type)
		if err != nil {
			if err == errTypeConversion {
				return nil, fmt.Errorf("cannot convert %v (type %s) to type %s", expr, t1, t2)
			}
			panic(tc.errorf(expr, "%s", err))
		}
		t1 = &TypeInfo{Type: t2.Type, Properties: PropertyIsConstant, Value: v}
	} else if t2.Untyped() {
		v, err := convertImplicit(t2, t1.Type)
		if err != nil {
			if err == errTypeConversion {
				panic(tc.errorf(expr, "cannot convert %v (type %s) to type %s", expr, t2, t1))
			}
			panic(tc.errorf(expr, "%s", err))
		}
		t2 = &TypeInfo{Type: t1.Type, Properties: PropertyIsConstant, Value: v}
	}

	if t1.IsConstant() && t2.IsConstant() {
		return tBinaryOp(t1, expr, t2)
	}

	if isComparison(expr.Op) {
		if !isAssignableTo(t1, t2.Type) && !isAssignableTo(t2, t1.Type) {
			panic(tc.errorf(expr, "invalid operation: %v (mismatched types %s and %s)", expr, t1.ShortString(), t2.ShortString()))
		}
		if expr.Op == ast.OperatorEqual || expr.Op == ast.OperatorNotEqual {
			if !t1.Type.Comparable() {
				// TODO(marco) explain in the error message why they are not comparable.
				panic(tc.errorf(expr, "invalid operation: %v (%s cannot be compared)", expr, t1.Type))
			}
		} else if !isOrdered(t1) {
			panic(tc.errorf(expr, "invalid operation: %v (operator %s not defined on %s)", expr, expr.Op, t1.Type.Kind()))
		}
		return &TypeInfo{Type: boolType, Properties: PropertyUntyped}, nil
	}

	if t1.Type != t2.Type {
		panic(tc.errorf(expr, "invalid operation: %v (mismatched types %s and %s)", expr, t1.ShortString(), t2.ShortString()))
	}

	if kind := t1.Type.Kind(); !operatorsOfKind[kind][expr.Op] {
		panic(tc.errorf(expr, "invalid operation: %v (operator %s not defined on %s)", expr, expr.Op, kind))
	}

	if t1.IsConstant() {
		return t2, nil
	}

	return t1, nil
}

// checkSize checks the type of expr as a make size parameter.
// If it is a constant returns the integer value, otherwise returns -1.
func (tc *typechecker) checkSize(expr ast.Expression, typ reflect.Type, name string) int {
	size := tc.checkExpression(expr)
	if size.Untyped() && !size.IsNumeric() || !size.Untyped() && !size.IsInteger() {
		got := size.String()
		if name == "size" {
			got = size.ShortString()
		}
		panic(tc.errorf(expr, "non-integer %s argument in make(%s) - %s", name, typ, got))
	}
	s := -1
	if size.IsConstant() {
		n, err := representedBy(size, intType)
		if err != nil {
			panic(tc.errorf(expr, fmt.Sprintf("%s", err)))
		}
		if s = int(n.(*big.Int).Int64()); s < 0 {
			panic(tc.errorf(expr, "negative %s argument in make(%s)", name, typ))
		}
	}
	return s
}

// checkCallBuiltin checks the builtin call expr, returning the list of results.
func (tc *typechecker) checkCallBuiltin(expr *ast.Call) []*TypeInfo {

	ident := expr.Func.(*ast.Identifier)

	if expr.IsVariadic && ident.Name != "append" {
		panic(tc.errorf(expr, "invalid use of ... with builtin %s", ident.Name))
	}

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
			panic(tc.errorf(expr, "first argument to append must be slice; have %s", slice.StringWithNumber(true)))
		}
		if expr.IsVariadic {
			if len(expr.Args) == 1 {
				panic(tc.errorf(expr, "cannot use ... on first argument to append"))
			} else if len(expr.Args) > 2 {
				panic(tc.errorf(expr, "too many arguments to append"))
			}
			t := tc.checkExpression(expr.Args[1])
			isSpecialCase := t.Type.Kind() == reflect.String && slice.Type.Elem() == uint8Type
			if !isSpecialCase && !isAssignableTo(t, slice.Type) {
				panic(tc.errorf(expr, "cannot use %s (type %s) as type %s in append", expr.Args[1], t, slice.Type))
			}
		} else if len(expr.Args) > 1 {
			elem := slice.Type.Elem()
			for _, el := range expr.Args[1:] {
				t := tc.checkExpression(el)
				if !isAssignableTo(t, elem) {
					if t == nil {
						panic(tc.errorf(expr, "cannot use nil as type %s in append", elem))
					}
					panic(tc.errorf(expr, "cannot use %s (type %s) as type %s in append", el, t.ShortString(), elem))
				}
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
		t := tc.checkExpression(expr.Args[0])
		if t.Nil() {
			panic(tc.errorf(expr, "use of untyped nil"))
		}
		switch k := t.Type.Kind(); k {
		case reflect.Slice, reflect.Array, reflect.Chan:
		default:
			if k != reflect.Ptr || t.Type.Elem().Kind() != reflect.Array {
				panic(tc.errorf(expr, "invalid argument %s (type %s) for cap", expr.Args[0], t.ShortString()))
			}
		}
		// TODO (Gianluca): «The expressions len(s) and cap(s) are constants
		// if the type of s is an array or pointer to an array and the
		// expression s does not contain channel receives or (non-constant)
		// function calls; in this case s is not evaluated.» (see
		// https://golang.org/ref/spec#Length_and_capacity).
		ti := &TypeInfo{Type: intType}
		if t.Type.Kind() == reflect.Array {
			ti.Properties = PropertyIsConstant
			ti.Value = int64(t.Type.Len())
		}
		if t.Type.Kind() == reflect.Ptr && t.Type.Elem().Kind() == reflect.Array {
			ti.Properties = PropertyIsConstant
			ti.Value = int64(t.Type.Elem().Len())
		}
		return []*TypeInfo{ti}

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
		t := tc.checkExpression(expr.Args[0])
		key := tc.checkExpression(expr.Args[1])
		if t.Nil() {
			panic(tc.errorf(expr, "first argument to delete must be map; have nil"))
		}
		if t.Type.Kind() != reflect.Map {
			panic(tc.errorf(expr, "first argument to delete must be map; have %s", t))
		}
		if !isAssignableTo(key, t.Type.Key()) {
			if key.Nil() {
				panic(tc.errorf(expr, "cannot use nil as type %s in delete", t.Type.Key()))
			}
			panic(tc.errorf(expr, "cannot use %v (type %s) as type %s in delete", expr.Args[1], key.ShortString(), t.Type.Key()))
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
		case reflect.String, reflect.Slice, reflect.Map, reflect.Array, reflect.Chan:
		default:
			if k != reflect.Ptr || t.Type.Elem().Kind() != reflect.Array {
				panic(tc.errorf(expr, "invalid argument %s (type %s) for len", expr.Args[0], t.ShortString()))
			}
		}
		ti := &TypeInfo{Type: intType}
		// TODO (Gianluca): «The expressions len(s) and cap(s) are constants
		// if the type of s is an array or pointer to an array and the
		// expression s does not contain channel receives or (non-constant)
		// function calls; in this case s is not evaluated.» (see
		// https://golang.org/ref/spec#Length_and_capacity).
		if t.IsConstant() && t.Type.Kind() == reflect.String {
			ti.Properties = PropertyIsConstant
			ti.Value = int64(len(t.Value.(string)))
		}
		if t.Type.Kind() == reflect.Array {
			ti.Properties = PropertyIsConstant
			ti.Value = int64(t.Type.Len())
		}
		if t.Type.Kind() == reflect.Ptr && t.Type.Elem().Kind() == reflect.Array {
			ti.Properties = PropertyIsConstant
			ti.Value = int64(t.Type.Elem().Len())
		}
		return []*TypeInfo{ti}

	case "make":
		numArgs := len(expr.Args)
		if numArgs == 0 {
			panic(tc.errorf(expr, "missing argument to make"))
		}
		t := tc.checkType(expr.Args[0], noEllipses)
		switch t.Type.Kind() {
		case reflect.Slice:
			if numArgs == 1 {
				panic(tc.errorf(expr, "missing len argument to make(%s)", expr.Args[0]))
			}
			if numArgs > 1 {
				l := tc.checkSize(expr.Args[1], t.Type, "len")
				if numArgs > 2 {
					c := tc.checkSize(expr.Args[2], t.Type, "cap")
					if l != -1 && c != -1 && l > c {
						panic(tc.errorf(expr, "len larger than cap in make(%s)", t.Type))
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
				_ = tc.checkSize(expr.Args[1], t.Type, "size")
			}
		default:
			panic(tc.errorf(expr, "cannot make type %s", t))
		}
		return []*TypeInfo{{Type: t.Type}}

	case "new":
		if len(expr.Args) == 0 {
			panic(tc.errorf(expr, "missing argument to new"))
		}
		t := tc.checkType(expr.Args[0], noEllipses)
		if len(expr.Args) > 1 {
			panic(tc.errorf(expr, "too many arguments to new(%s)", expr.Args[0]))
		}
		return []*TypeInfo{{Type: reflect.PtrTo(t.Type)}}

	case "panic":
		if len(expr.Args) == 0 {
			panic(tc.errorf(expr, "missing argument to panic: panic()"))
		}
		if len(expr.Args) > 1 {
			panic(tc.errorf(expr, "too many arguments to panic: %s", expr))
		}
		_ = tc.checkExpression(expr.Args[0])
		return nil

	case "print", "println":
		for _, arg := range expr.Args {
			_ = tc.checkExpression(arg)
		}
		return nil

	}

	panic(fmt.Sprintf("unexpected builtin %s", ident.Name))

}

// checkCallExpression type checks a call expression, including type conversions
// and built-in function calls. Returns a list of typeinfos obtained from the
// call and a boolean value indicating if expr is a builtin.
func (tc *typechecker) checkCallExpression(expr *ast.Call, statement bool) ([]*TypeInfo, bool) {

	if ident, ok := expr.Func.(*ast.Identifier); ok {
		if t, ok := tc.lookupScopes(ident.Name, false); ok && t == builtinTypeInfo {
			return tc.checkCallBuiltin(expr), true
		}
	}

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
		value, err := convert(arg, t.Type)
		if err != nil {
			if err == errTypeConversion {
				panic(tc.errorf(expr, "cannot convert %s (type %s) to type %s", expr.Args[0], arg.Type, t.Type))
			}
			panic(tc.errorf(expr, "%s", err))
		}
		ti := &TypeInfo{Type: t.Type, Value: value}
		if value != nil {
			ti.Properties = PropertyIsConstant
		}
		return []*TypeInfo{ti}, false
	}

	if t.Type.Kind() != reflect.Func {
		panic(tc.errorf(expr, "cannot call non-function %v (type %s)", expr.Func, t))
	}

	var variadic = t.Type.IsVariadic()

	if expr.IsVariadic && !variadic {
		panic(tc.errorf(expr, "invalid use of ... in call to %s", expr.Func))
	}

	var numIn = t.Type.NumIn()

	args := expr.Args

	if len(args) == 1 && numIn > 1 {
		if c, ok := args[0].(*ast.Call); ok {
			args = nil
			tis, _ := tc.checkCallExpression(c, false)
			for _, ti := range tis {
				v := ast.NewCall(c.Pos(), c.Func, c.Args, false)
				tc.typeInfo[v] = ti
				args = append(args, v)
			}
		}
	}

	if (!variadic && len(args) != numIn) || (variadic && len(args) < numIn-1) {
		have := "("
		for i, arg := range args {
			if i > 0 {
				have += ", "
			}
			c := tc.typeInfo[arg]
			if c == nil {
				c = tc.checkExpression(arg)
			}
			if c == nil {
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
			if i == numIn-1 && variadic {
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

	var lastIn = numIn - 1
	var in reflect.Type

	for i, arg := range args {
		if i < lastIn || !variadic {
			in = t.Type.In(i)
		} else if i == lastIn {
			in = t.Type.In(lastIn).Elem()
		}
		a := tc.typeInfo[arg]
		if a == nil {
			a = tc.checkExpression(arg)
		}
		if expr.IsVariadic && i == lastIn {
			if a.Type.Kind() != reflect.Slice && a.Type.Kind() != reflect.Array {
				if variadic {
					in = reflect.SliceOf(in)
				}
				panic(tc.errorf(args[i], "cannot use %s (type %s) as type %s in argument to %s", args[i], a.ShortString(), in, expr.Func))
			}
			a.Type = a.Type.Elem()
			if !isAssignableTo(a, in) {
				if variadic {
					in = reflect.SliceOf(in)
				}
				a.Type = reflect.SliceOf(a.Type)
				panic(tc.errorf(args[i], "cannot use %s (type %s) as type %s in argument to %s", args[i], a.ShortString(), in, expr.Func))
			}
		} else {
			if !isAssignableTo(a, in) {
				panic(tc.errorf(args[i], "cannot use %s (type %s) as type %s in argument to %s", args[i], a.ShortString(), in, expr.Func))
			}
		}
	}

	numOut := t.Type.NumOut()
	resultTypes := make([]*TypeInfo, numOut)
	for i := 0; i < numOut; i++ {
		resultTypes[i] = &TypeInfo{Type: t.Type.Out(i)}
	}

	return resultTypes, false
}
