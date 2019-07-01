// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"errors"
	"fmt"
	"reflect"
	"unicode"

	"scriggo/internal/compiler/ast"
	"scriggo/vm"
)

const noEllipses = -1

type scopeElement struct {
	t    *TypeInfo
	decl *ast.Identifier
}

type TypeCheckerScope map[string]scopeElement

type HTML string

var boolType = reflect.TypeOf(false)
var uintType = reflect.TypeOf(uint(0))
var uint8Type = reflect.TypeOf(uint8(0))
var int32Type = reflect.TypeOf(int32(0))

var builtinTypeInfo = &TypeInfo{Properties: PropertyIsBuiltin}
var uint8TypeInfo = &TypeInfo{Type: uint8Type, Properties: PropertyIsType}
var int32TypeInfo = &TypeInfo{Type: int32Type, Properties: PropertyIsType}

var untypedBoolTypeInfo = &TypeInfo{Type: boolType, Properties: PropertyUntyped}

var envType = reflect.TypeOf(&vm.Env{})

var universe = TypeCheckerScope{
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
	"complex128":  {t: &TypeInfo{Type: complex128Type, Properties: PropertyIsType}},
	"complex64":   {t: &TypeInfo{Type: complex64Type, Properties: PropertyIsType}},
	"error":       {t: &TypeInfo{Type: reflect.TypeOf((*error)(nil)).Elem(), Properties: PropertyIsType}},
	"float32":     {t: &TypeInfo{Type: reflect.TypeOf(float32(0)), Properties: PropertyIsType}},
	"float64":     {t: &TypeInfo{Type: float64Type, Properties: PropertyIsType}},
	"false":       {t: &TypeInfo{Type: boolType, Properties: PropertyUntyped, Constant: boolConst(false)}},
	"int":         {t: &TypeInfo{Type: intType, Properties: PropertyIsType}},
	"int16":       {t: &TypeInfo{Type: reflect.TypeOf(int16(0)), Properties: PropertyIsType}},
	"int32":       {t: int32TypeInfo},
	"int64":       {t: &TypeInfo{Type: reflect.TypeOf(int64(0)), Properties: PropertyIsType}},
	"int8":        {t: &TypeInfo{Type: reflect.TypeOf(int8(0)), Properties: PropertyIsType}},
	"interface{}": {t: &TypeInfo{Type: emptyInterfaceType, Properties: PropertyIsType}},
	"rune":        {t: int32TypeInfo},
	"string":      {t: &TypeInfo{Type: stringType, Properties: PropertyIsType}},
	"true":        {t: &TypeInfo{Type: boolType, Properties: PropertyUntyped, Constant: boolConst(true)}},
	"uint":        {t: &TypeInfo{Type: uintType, Properties: PropertyIsType}},
	"uint16":      {t: &TypeInfo{Type: reflect.TypeOf(uint16(0)), Properties: PropertyIsType}},
	"uint32":      {t: &TypeInfo{Type: reflect.TypeOf(uint32(0)), Properties: PropertyIsType}},
	"uint64":      {t: &TypeInfo{Type: reflect.TypeOf(uint64(0)), Properties: PropertyIsType}},
	"uint8":       {t: uint8TypeInfo},
	"uintptr":     {t: &TypeInfo{Type: reflect.TypeOf(uintptr(0)), Properties: PropertyIsType}},
}

type ancestor struct {
	scopeLevel int
	node       ast.Node
}

type scopeVariable struct {
	ident      string
	scopeLevel int
	node       ast.Node
}

// typechecker represents the state of a type checking.
type typechecker struct {
	path              string
	predefinedPkgs    map[string]*Package
	Universe          TypeCheckerScope
	filePackageBlock  TypeCheckerScope
	Scopes            []TypeCheckerScope
	ancestors         []*ancestor
	terminating       bool // https://golang.org/ref/spec#Terminating_statements
	hasBreak          map[ast.Node]bool
	unusedVars        []*scopeVariable
	unusedImports     map[string][]string
	TypeInfo          map[ast.Node]*TypeInfo
	IndirectVars      map[*ast.Identifier]bool
	opts              Options
	iota              int
	lastConstPosition *ast.Position    // when changes iota is reset.
	showMacros        []*ast.ShowMacro // list of *ast.ShowMacro nodes.

	// Data structures for Goto and Labels checking.
	gotos           []string
	storedGotos     []string
	nextValidGoto   int
	storedValidGoto int
	labels          [][]string
}

func newTypechecker(path string, opts Options) *typechecker {
	return &typechecker{
		path:             path,
		filePackageBlock: TypeCheckerScope{},
		hasBreak:         map[ast.Node]bool{},
		predefinedPkgs:   map[string]*Package{},
		TypeInfo:         map[ast.Node]*TypeInfo{},
		Universe:         universe,
		unusedImports:    map[string][]string{},
		IndirectVars:     map[*ast.Identifier]bool{},
		opts:             opts,
		iota:             -1,
	}
}

// addScope adds a new empty scope to the type checker.
func (tc *typechecker) addScope() {
	tc.Scopes = append(tc.Scopes, TypeCheckerScope{})
	tc.labels = append(tc.labels, []string{})
	tc.storedGotos = tc.gotos
	tc.gotos = []string{}
}

// removeCurrentScope removes the current scope from the type checker.
func (tc *typechecker) removeCurrentScope() {
	if !tc.opts.IsScript && !tc.opts.IsTemplate {
		cut := len(tc.unusedVars)
		for i := len(tc.unusedVars) - 1; i >= 0; i-- {
			v := tc.unusedVars[i]
			if v.scopeLevel < len(tc.Scopes)-1 {
				break
			}
			if v.node != nil {
				panic(tc.errorf(v.node, "%s declared and not used", v.ident))

			}
			cut = i
		}
		tc.unusedVars = tc.unusedVars[:cut]
	}
	tc.Scopes = tc.Scopes[:len(tc.Scopes)-1]
	tc.labels = tc.labels[:len(tc.labels)-1]
	tc.gotos = append(tc.storedGotos, tc.gotos...)
	tc.nextValidGoto = tc.storedValidGoto
}

// lookupScopesElem looks up name in the scopes. Returns the declaration of the
// name or false if the name does not exist. If justCurrentScope is true,
// lookupScopesElem looks up only in the current scope.
func (tc *typechecker) lookupScopesElem(name string, justCurrentScope bool) (scopeElement, bool) {
	// Iterating over scopes, from inside.
	for i := len(tc.Scopes) - 1; i >= 0; i-- {
		elem, ok := tc.Scopes[i][name]
		if ok {
			return elem, true
		}
		if justCurrentScope && i == len(tc.Scopes)-1 {
			return scopeElement{}, false
		}
	}
	// Package + file block.
	if elem, ok := tc.filePackageBlock[name]; ok {
		return elem, true
	}
	// Universe.
	if elem, ok := tc.Universe[name]; ok {
		return elem, true
	}
	return scopeElement{}, false
}

// lookupScopes looks up name in the scopes. Returns the type info of the name or
// false if the name does not exist. If justCurrentScope is true, lookupScopes
// looks up only in the current scope.
func (tc *typechecker) lookupScopes(name string, justCurrentScope bool) (*TypeInfo, bool) {
	elem, ok := tc.lookupScopesElem(name, justCurrentScope)
	if !ok {
		return nil, false
	}
	return elem.t, ok
}

// assignScope assigns value to name in the last scope. If there are no scopes,
// value is assigned in file/package block.
func (tc *typechecker) assignScope(name string, value *TypeInfo, declNode *ast.Identifier) {
	if len(tc.Scopes) == 0 {
		if _, ok := tc.filePackageBlock[name]; ok {
			panic("redeclared in this block...") // TODO(Gianluca): to review.
		}
		tc.filePackageBlock[name] = scopeElement{t: value, decl: declNode}
	} else {
		tc.Scopes[len(tc.Scopes)-1][name] = scopeElement{t: value, decl: declNode}
	}
}

func (tc *typechecker) addToAncestors(n ast.Node) {
	tc.ancestors = append(tc.ancestors, &ancestor{len(tc.Scopes), n})
}

func (tc *typechecker) removeLastAncestor() {
	tc.ancestors = tc.ancestors[:len(tc.ancestors)-1]
}

// currentFunction returns the current function and the related scope level.
// If it is called when not in a function body, returns nil and 0.
func (tc *typechecker) currentFunction() (*ast.Func, int) {
	for i := len(tc.ancestors) - 1; i >= 0; i-- {
		if f, ok := tc.ancestors[i].node.(*ast.Func); ok {
			return f, tc.ancestors[i].scopeLevel
		}
	}
	return nil, 0
}

// isUpValue checks if name is an upvalue.
func (tc *typechecker) isUpValue(name string) bool {
	_, funcBound := tc.currentFunction()
	for i := len(tc.Scopes) - 1; i >= 0; i-- {
		for n := range tc.Scopes[i] {
			if n != name {
				continue
			}
			if i < funcBound-1 { // out of current function scope.
				tc.IndirectVars[tc.Scopes[i][n].decl] = true
				return true
			}
			return false
		}
	}
	return false
}

// getScopeLevel returns the scope level in which name is declared, and a
// boolean which reports whether has been imported from another package/page
// or not.
func (tc *typechecker) getScopeLevel(name string) (int, bool) {
	// Iterating over scopes, from inside.
	for i := len(tc.Scopes) - 1; i >= 0; i-- {
		if _, ok := tc.Scopes[i][name]; ok {
			return i + 1, false // TODO(Gianluca): to review.
		}
	}
	return -1, tc.filePackageBlock[name].decl == nil
}

// getDeclarationNode returns the declaration node which declares name.
func (tc *typechecker) getDeclarationNode(name string) ast.Node {
	// Iterating over scopes, from inside.
	for i := len(tc.Scopes) - 1; i >= 0; i-- {
		elem, ok := tc.Scopes[i][name]
		if ok {
			return elem.decl
		}
	}
	// Package + file block.
	if elem, ok := tc.filePackageBlock[name]; ok {
		return elem.decl
	}
	// Universe.
	if elem, ok := tc.Universe[name]; ok {
		return elem.decl
	}
	panic(fmt.Sprintf("trying to get scope level of %s, but any scope, package block, file block or universe contains it", name)) // TODO(Gianluca): to review.
}

// funcChain returns a list of ordered functions from the brother of name's
// declaration to the inner one.
func (tc *typechecker) funcChain(name string) []*ast.Func {
	funcs := []*ast.Func{}
	declLevel, imported := tc.getScopeLevel(name)
	// If name has been imported, function chain does not exist.
	if imported {
		return nil
	}
	for _, anc := range tc.ancestors {
		if fun, ok := anc.node.(*ast.Func); ok {
			if declLevel < anc.scopeLevel {
				funcs = append(funcs, fun)
			}
		}
	}
	return funcs
}

// isPackageVariable reports whether name is a package variable.
func (tc *typechecker) isPackageVariable(name string) bool {
	_, ok := tc.filePackageBlock[name]
	return ok
}

// showMacroIgnoredTi is the TypeInfo of a ShowMacro identifier which is
// undefined but has been marked as to be ignored or "todo".
var showMacroIgnoredTi = &TypeInfo{}

// checkIdentifier checks identifier ident, returning it's typeinfo retrieved
// from scope. If using, ident is marked as "used".
func (tc *typechecker) checkIdentifier(ident *ast.Identifier, using bool) *TypeInfo {

	if tc.isUpValue(ident.Name) || tc.isPackageVariable(ident.Name) {
		decl := tc.getDeclarationNode(ident.Name)
		upvar := ast.Upvar{decl, -1}
		chain := tc.funcChain(ident.Name)
		for _, f := range chain {
			contains := false
			for i, uv := range f.Upvars {
				if uv.Declaration == upvar.Declaration {
					contains = true
					upvar.Index = int16(i)
					break
				}
			}
			if !contains {
				f.Upvars = append(f.Upvars, upvar)
				upvar.Index = int16(len(f.Upvars) - 1)
			}
		}
	}

	// Looks for upvalues.
	if fun, _ := tc.currentFunction(); fun != nil {
		if tc.isUpValue(ident.Name) {
			fun.Upvalues = append(fun.Upvalues, ident.Name)
		}
	}

	i, ok := tc.lookupScopes(ident.Name, false)
	if !ok {
		if ident.Name == "iota" && tc.iota >= 0 {
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
		// TODO(Gianluca): add support for "or todo" error when typechecking
		// with "todosNotAllowed" option.
		for _, sm := range tc.showMacros {
			if sm.Macro == ident && (sm.Or == ast.ShowMacroOrIgnore || sm.Or == ast.ShowMacroOrTodo) {
				return showMacroIgnoredTi
			}
		}
		panic(tc.errorf(ident, "undefined: %s", ident.Name))
	}

	if i == builtinTypeInfo {
		panic(tc.errorf(ident, "use of builtin %s not in function call", ident.Name))
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

	return i
}

// CheckingError records a typechecking error with the path and the position where the
// error occurred.
type CheckingError struct {
	Path string
	Pos  ast.Position
	Err  error
}

func (e *CheckingError) Error() string {
	return fmt.Sprintf("%s:%s: %s", e.Path, e.Pos, e.Err)
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
	var err = &CheckingError{
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
	tc.TypeInfo[expr] = ti
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
		tc.TypeInfo[expr] = newTi
		return newTi
	}
	ti := tc.typeof(expr, length)
	if !ti.IsType() {
		panic(tc.errorf(expr, "%s is not a type", expr))
	}
	tc.TypeInfo[expr] = ti
	return ti
}

// typeof returns the type of expr. If expr is not an expression but a type,
// returns the type.
func (tc *typechecker) typeof(expr ast.Expression, length int) *TypeInfo {

	// TODO: remove double type check
	ti := tc.TypeInfo[expr]
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
		return &TypeInfo{
			Type:       typ,
			Properties: PropertyUntyped,
			Constant:   parseBasicLiteral(expr.Type, expr.Value),
		}

	case *ast.Parenthesis:
		panic("unexpected parenthesis")

	case *ast.UnaryOperator:
		t := tc.checkExpression(expr.Expr)
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
				panic(tc.errorf(expr, "invalid indirect of %s (type %s)", expr.Expr, t))
			}
			ti.Type = t.Type.Elem()
			ti.Properties = ti.Properties | PropertyAddressable
		case ast.OperatorAnd:
			if _, ok := expr.Expr.(*ast.CompositeLiteral); !ok && !t.Addressable() {
				panic(tc.errorf(expr, "cannot take the address of %s", expr.Expr))
			}
			ti.Type = reflect.PtrTo(t.Type)
			// When taking the address of a variable, such variable must be
			// marked as "indirect".
			if ident, ok := expr.Expr.(*ast.Identifier); ok {
			scopesLoop:
				for i := len(tc.Scopes) - 1; i >= 0; i-- {
					for n := range tc.Scopes[i] {
						if n == ident.Name {
							tc.IndirectVars[tc.Scopes[i][n].decl] = true
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
				panic(tc.errorf(expr, "invalid operation: %s (receive from non-chan type %s)", expr.Expr, t.Type))
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
		for _, fd := range expr.FieldDecl {
			typ := tc.checkType(fd.Type, noEllipses).Type
			if fd.IdentifierList == nil {
				// Implicit field declaration.
				fields = append(fields, reflect.StructField{
					Name:      "Name", // TODO (Gianluca): to review.
					PkgPath:   "",     // TODO (Gianluca): to review.
					Type:      typ,
					Tag:       "",  // TODO (Gianluca): to review.
					Offset:    0,   // TODO (Gianluca): to review.
					Index:     nil, // TODO (Gianluca): to review.
					Anonymous: true,
				})
			} else {
				// Explicit field declaration.
				for _, ident := range fd.IdentifierList {
					fields = append(fields, reflect.StructField{
						Name:      ident.Name,
						PkgPath:   "", // TODO (Gianluca): to review.
						Type:      typ,
						Tag:       "",  // TODO (Gianluca): to review.
						Offset:    0,   // TODO (Gianluca): to review.
						Index:     nil, // TODO (Gianluca): to review.
						Anonymous: false,
					})
				}
			}
		}
		t := reflect.StructOf(fields)
		return &TypeInfo{
			Type:       t,
			Properties: PropertyIsType,
		}

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
		ti := tc.checkExpression(expr.Len)
		if !ti.IsConstant() {
			panic(tc.errorf(expr, "non-constant array bound %s", expr.Len))
		}
		c, err := convert(ti, intType)
		if err != nil {
			panic(tc.errorf(expr, "%s", err))
		}
		b := int(c.int64())
		if b < 0 {
			panic(tc.errorf(expr, "array bound must be non-negative"))
		}
		if b < length {
			panic(tc.errorf(expr, "array index %d out of bounds [0:%d]", length-1, b))
		}
		return &TypeInfo{Properties: PropertyIsType, Type: reflect.ArrayOf(b, elem.Type)}

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
		elem := tc.checkType(expr.ElementType, noEllipses)
		return &TypeInfo{Properties: PropertyIsType, Type: reflect.ChanOf(dir, elem.Type)}

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
		expr.Reflect = reflect.FuncOf(in, out, variadic)
		return &TypeInfo{Type: expr.Reflect, Properties: PropertyIsType}

	case *ast.Func:
		tc.addScope()
		t := tc.checkType(expr.Type, noEllipses)
		expr.Type.Reflect = t.Type
		tc.ancestors = append(tc.ancestors, &ancestor{len(tc.Scopes), expr})
		// Adds parameters to the function body scope.
		fillParametersTypes(expr.Type.Parameters)
		isVariadic := expr.Type.IsVariadic
		for i, f := range expr.Type.Parameters {
			t := tc.checkType(f.Type, noEllipses)
			if f.Ident != nil {
				if isVariadic && i == len(expr.Type.Parameters)-1 {
					tc.assignScope(f.Ident.Name, &TypeInfo{Type: reflect.SliceOf(t.Type), Properties: PropertyAddressable}, nil)
					continue
				}
				tc.assignScope(f.Ident.Name, &TypeInfo{Type: t.Type, Properties: PropertyAddressable}, nil)
			}
		}
		// Adds named return values to the function body scope.
		fillParametersTypes(expr.Type.Result)
		for _, f := range expr.Type.Result {
			t := tc.checkType(f.Type, noEllipses)
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
		tc.removeCurrentScope()
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
			tc.checkIndex(expr.Index, t, false)
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
			if err := isAssignableTo(key, expr.Index, t.Type.Key()); err != nil {
				if _, ok := err.(invalidTypeInAssignment); ok {
					panic(tc.errorf(expr, "%s in map index", err))
				}
				panic(tc.errorf(expr, "%s", err))
			}
			key.SetValue(t.Type.Key())
			return &TypeInfo{Type: t.Type.Elem()}
		default:
			panic(tc.errorf(expr, "invalid operation: %s (type %s does not support indexing)", expr, t.ShortString()))
		}

	case *ast.Slicing:
		t := tc.checkExpression(expr.Expr)
		if t.Nil() {
			panic(tc.errorf(expr, "use of untyped nil"))
		}
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
					pkg := ti.value.(*PackageInfo)
					v, ok := pkg.Declarations[expr.Ident]
					if !ok {
						// If identifiers is a ShowMacro identifier, first needs to check if
						// ShowMacro contains a "or ignore" or "or todo" option. In such cases,
						// error should not be returned, and function call should be removed
						// from tree.
						// TODO(Gianluca): add support for "or todo" error when typechecking
						// with "todosNotAllowed" option.
						for _, sm := range tc.showMacros {
							if sm.Macro == ident && (sm.Or == ast.ShowMacroOrIgnore || sm.Or == ast.ShowMacroOrTodo) {
								return showMacroIgnoredTi
							}
						}
						panic(tc.errorf(expr, "undefined: %v", expr))
					}
					tc.TypeInfo[expr] = v
					return v
				}
			}
		}
		t := tc.typeof(expr.Expr, noEllipses)
		tc.TypeInfo[expr.Expr] = t
		// t is a type.
		if t.IsType() {
			method, _, ok := methodByName(t, expr.Ident)
			if !ok {
				panic(tc.errorf(expr, "%v undefined (type %s has no method %s)", expr, t, expr.Ident))
			}
			return method
		}
		// TODO(Gianluca): is discriminating pointers necessary?
		// // t is a pointer.
		// if t.Type.Kind() == reflect.Ptr {
		// 	method, _, ok := methodByName(t, expr.Ident)
		// 	if ok {
		// 		return method
		// 	}
		// 	field, ok := fieldByName(t, expr.Ident)
		// 	if ok {
		// 		return field
		// 	}
		// 	panic(tc.errorf(expr, "%v undefined (type %s has no field or method %s)", expr, t, expr.Ident))
		// }
		// regular case.
		method, trans, ok := methodByName(t, expr.Ident)
		if ok {
			switch trans {
			case receiverAddAddress:
				if t.Addressable() {
					elem, _ := tc.lookupScopesElem(expr.Expr.(*ast.Identifier).Name, false)
					tc.IndirectVars[elem.decl] = true
					expr.Expr = ast.NewUnaryOperator(expr.Pos(), ast.OperatorAnd, expr.Expr)
					tc.TypeInfo[expr.Expr] = &TypeInfo{
						Type:       reflect.PtrTo(t.Type),
						MethodType: t.MethodType,
					}
				}
			case receiverAddIndirect:
				expr.Expr = ast.NewUnaryOperator(expr.Pos(), ast.OperatorMultiplication, expr.Expr)
				tc.TypeInfo[expr.Expr] = &TypeInfo{
					Type:       t.Type.Elem(),
					MethodType: t.MethodType,
				}
			}
			return method
		}
		field, ok := fieldByName(t, expr.Ident)
		if ok {
			return field
		}
		panic(tc.errorf(expr, "%v undefined (type %s has no field or method %s)", expr, t, expr.Ident))

	case *ast.TypeAssertion:
		t := tc.checkExpression(expr.Expr)
		if t.Type.Kind() != reflect.Interface {
			panic(tc.errorf(expr, "invalid type assertion: %v (non-interface type %s on left)", expr, t))
		}
		typ := tc.checkType(expr.Type, noEllipses)
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
	index := tc.checkExpression(expr)
	if index.Untyped() && !index.IsNumeric() || !index.Untyped() && !index.IsInteger() {
		if isSlice {
			panic(tc.errorf(expr, "invalid slice index %s (type %s)", expr, index))
		}
		panic(tc.errorf(expr, "non-integer %s index %s", typ.Kind(), expr))
	}
	index.SetValue(intType)
	if index.IsConstant() {
		c, err := convert(index, intType)
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

	t1 := tc.checkExpression(expr1)
	t2 := tc.checkExpression(expr2)

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
			} else {
				_, err = newIntConst(0).binaryOp(op, t2.Constant)
			}
			if err != nil {
				switch err {
				case errNegativeShiftCount:
					err = fmt.Errorf("invalid negative shift count: %s", t2.Constant)
				case errShiftCountTooLarge:
					err = fmt.Errorf("shift count too large: %s", t2.Constant)
				case errShiftCountTruncatedToInteger:
					err = fmt.Errorf("constant %s truncated to integer", t2.Constant)
				}
				return nil, err
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
				ti.Constant, err = convert(ti, ti.Type)
				if err != nil {
					return nil, fmt.Errorf("constant %v overflows %s", c, t1)
				}
			}
		}
		return ti, nil
	}

	if t1.Nil() || t2.Nil() {
		if t1.Nil() && t2.Nil() {
			return nil, fmt.Errorf("operator %s not defined on nil", op)
		}
		t := t1
		if t.Nil() {
			t = t2
		}
		k := t.Type.Kind()
		if reflect.Bool <= k && k <= reflect.Array || k == reflect.String || k == reflect.Struct {
			return nil, fmt.Errorf("cannot convert nil to type %s", t.ShortString())
		}
		if op != ast.OperatorEqual && op != ast.OperatorNotEqual {
			return nil, fmt.Errorf("operator %s not defined on %s", op, k)
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

	if t1.IsUntypedConstant() {
		c, err := representedBy(t1, t2.Type)
		if err != nil {
			return nil, err
		}
		t1.SetValue(t2.Type)
		t1 = &TypeInfo{Type: t2.Type, Constant: c}
	} else if t2.IsUntypedConstant() {
		c, err := representedBy(t2, t1.Type)
		if err != nil {
			return nil, err
		}
		t2.SetValue(t1.Type)
		t2 = &TypeInfo{Type: t1.Type, Constant: c}
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
			return &TypeInfo{Type: boolType, Constant: c}, nil
		}
		ti := &TypeInfo{Type: t1.Type, Constant: c}
		ti.Constant, err = convert(ti, t1.Type)
		if err != nil {
			return nil, fmt.Errorf("constant %v overflows %s", c, t1)
		}
		return ti, nil
	}

	if isComparison(op) {
		if isAssignableTo(t1, expr1, t2.Type) != nil && isAssignableTo(t2, expr2, t1.Type) != nil {
			return nil, fmt.Errorf("mismatched types %s and %s", t1.ShortString(), t2.ShortString())
		}
		if op == ast.OperatorEqual || op == ast.OperatorNotEqual {
			if !t1.Type.Comparable() {
				// TODO(marco) explain in the error message why they are not comparable.
				return nil, fmt.Errorf("%s cannot be compared", t1.Type)
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

	if t1.IsConstant() {
		return t2, nil
	}

	return t1, nil
}

// checkSize checks the type of expr as a make size parameter.
// If it is a constant returns the integer value, otherwise returns -1.
func (tc *typechecker) checkSize(expr ast.Expression, typ reflect.Type, name string) constant {
	size := tc.checkExpression(expr)
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
	if size.IsConstant() {
		c, err := convert(size, intType)
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
			if !isSpecialCase && isAssignableTo(t, expr.Args[1], slice.Type) != nil {
				panic(tc.errorf(expr, "cannot use %s (type %s) as type %s in append", expr.Args[1], t, slice.Type))
			}
		} else if len(expr.Args) > 1 {
			elemType := slice.Type.Elem()
			for i, el := range expr.Args {
				if i == 0 {
					continue
				}
				t := tc.checkExpression(el)
				if err := isAssignableTo(t, el, elemType); err != nil {
					if _, ok := err.(invalidTypeInAssignment); ok {
						panic(tc.errorf(expr, "%s in append", err))
					}
					panic(tc.errorf(expr, "%s", err))
				}
				t.SetValue(elemType)
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
			ti.Constant = int64Const(t.Type.Len())
		}
		if t.Type.Kind() == reflect.Ptr && t.Type.Elem().Kind() == reflect.Array {
			ti.Constant = int64Const(t.Type.Elem().Len())
		}
		return []*TypeInfo{ti}

	case "close":
		// TODO(Gianluca): add specific "close" errors.
		tc.checkExpression(expr.Args[0])
		return []*TypeInfo{}

	case "complex":
		// TODO(Gianluca): add SetValue.
		switch len(expr.Args) {
		case 0:
			panic(tc.errorf(expr, "missing argument to complex - complex(<N>, <N>)"))
		case 1:
			panic(tc.errorf(expr, "invalid operation: complex expects two arguments"))
		case 2:
		default:
			panic(tc.errorf(expr, "too many arguments to complex - complex(%s, <N>)", expr.Args[0]))
		}
		re := tc.checkExpression(expr.Args[0])
		im := tc.checkExpression(expr.Args[1])
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
			c, err := convert(re, im.Type)
			if err != nil {
				panic(tc.errorf(expr, "cannot convert %s (type %s) to type %s", re.Constant, re, im))
			}
			re = &TypeInfo{Type: im.Type, Constant: c}
		} else if im.IsUntypedConstant() {
			c, err := convert(im, re.Type)
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
		keyType := t.Type.Key()
		if err := isAssignableTo(key, expr.Args[1], keyType); err != nil {
			if _, ok := err.(invalidTypeInAssignment); ok {
				panic(tc.errorf(expr, "%s in delete", err))
			}
			panic(tc.errorf(expr, "%s", err))
		}
		if key.IsConstant() {
			_, err := convert(key, keyType)
			if err != nil {
				panic(tc.errorf(expr, "%s", err))
			}
		}
		key.SetValue(keyType)
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
		t.SetValue(nil)
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
			ti.Constant = int64Const(len(t.Constant.string()))
		}
		if t.Type.Kind() == reflect.Array {
			ti.Constant = int64Const(t.Type.Len())
		}
		if t.Type.Kind() == reflect.Ptr && t.Type.Elem().Kind() == reflect.Array {
			ti.Constant = int64Const(t.Type.Elem().Len())
		}
		ti.SetValue(nil)
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
		ti := tc.checkExpression(expr.Args[0])
		ti.SetValue(nil)
		return nil

	case "print", "println":
		for _, arg := range expr.Args {
			tc.checkExpression(arg)
			tc.TypeInfo[arg].SetValue(nil)
		}
		return nil

	case "real", "imag":
		// TODO(Gianluca): add SetValue.
		switch len(expr.Args) {
		case 0:
			panic(tc.errorf(expr, "missing argument to %s: %s()", ident.Name, ident.Name))
		case 1:
		default:
			panic(tc.errorf(expr, "too many arguments to %s: %s", ident.Name, expr))
		}
		t := tc.checkExpression(expr.Args[0])
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

	if ident, ok := expr.Func.(*ast.Identifier); ok {
		if t, ok := tc.lookupScopes(ident.Name, false); ok && t == builtinTypeInfo {
			tc.TypeInfo[expr.Func] = t
			return tc.checkBuiltinCall(expr), true, false
		}
	}

	t := tc.typeof(expr.Func, noEllipses)

	switch t.MethodType {
	case MethodValueConcrete:
		t.MethodType = MethodCallConcrete
	case MethodValueInterface:
		t.MethodType = MethodCallInterface
	}

	tc.TypeInfo[expr.Func] = t

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
		arg := tc.checkExpression(expr.Args[0])
		c, err := convert(arg, t.Type)
		if err != nil {
			if err == errTypeConversion {
				panic(tc.errorf(expr, "cannot convert %s (type %s) to type %s", expr.Args[0], arg.Type, t.Type))
			}
			panic(tc.errorf(expr, "%s", err))
		}
		arg.SetValue(t.Type)
		return []*TypeInfo{{Type: t.Type, Constant: c}}, false, true
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
			for _, ti := range tis {
				v := ast.NewCall(c.Pos(), c.Func, c.Args, false)
				tc.TypeInfo[v] = ti
				args = append(args, v)
			}
		}
	} else if len(args) == 1 && numIn == 1 && funcIsVariadic && !callIsVariadic {
		if c, ok := args[0].(*ast.Call); ok {
			args = nil
			tis, _, _ := tc.checkCallExpression(c, false)
			for _, ti := range tis {
				v := ast.NewCall(c.Pos(), c.Func, c.Args, false)
				tc.TypeInfo[v] = ti
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
			c := tc.TypeInfo[arg]
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
			a := tc.TypeInfo[arg]
			if err := isAssignableTo(a, arg, in); err != nil {
				if _, ok := err.(invalidTypeInAssignment); ok {
					panic(tc.errorf(args[i], "cannot use %s as type %s in argument to %s", a, in, expr.Func))
				}
				panic(tc.errorf(expr, "%s", err))
			}
			continue
		}
		a := tc.checkExpression(arg)
		if i == lastIn && callIsVariadic {
			if err := isAssignableTo(a, arg, reflect.SliceOf(in)); err != nil {
				if _, ok := err.(invalidTypeInAssignment); ok {
					panic(tc.errorf(expr, "%s in argument to %s", err, expr.Func))
				}
				panic(tc.errorf(expr, "%s", err))
			}
			continue
		}
		if err := isAssignableTo(a, arg, in); err != nil {
			if _, ok := err.(invalidTypeInAssignment); ok {
				panic(tc.errorf(expr, "%s in argument to %s", err, expr.Func))
			}
			panic(tc.errorf(expr, "%s", err))
		}
		if a.Nil() {
			a.value = reflect.Zero(in).Interface()
			tc.TypeInfo[expr.Args[i]].Type = in
		}
		a.SetValue(in)
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
			ti := tc.checkExpression(kv.Key)
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

	maxIndex := -1
	switch node.Type.(type) {
	case *ast.ArrayType, *ast.SliceType:
		maxIndex = tc.maxIndex(node)
	}

	ti := tc.checkType(node.Type, maxIndex+1)

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
				fieldTi, ok := ti.Type.FieldByName(ident.Name)
				if !ok {
					panic(tc.errorf(node, "unknown field '%s' in struct literal of type %s", keyValue.Key, ti))
				}
				valueTi := tc.checkExpression(keyValue.Value)
				if err := isAssignableTo(valueTi, keyValue.Value, fieldTi.Type); err != nil {
					if _, ok := err.(invalidTypeInAssignment); ok {
						panic(tc.errorf(node, "%s in field value", err))
					}
					panic(tc.errorf(node, "%s", err))
				}
				valueTi.SetValue(fieldTi.Type)
			}
		case false: // struct with implicit fields.
			if len(node.KeyValues) == 0 {
				ti := &TypeInfo{Type: ti.Type}
				tc.TypeInfo[node] = ti
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
				valueTi := tc.checkExpression(keyValue.Value)
				fieldTi := ti.Type.Field(i)
				if err := isAssignableTo(valueTi, keyValue.Value, fieldTi.Type); err != nil {
					if _, ok := err.(invalidTypeInAssignment); ok {
						panic(tc.errorf(node, "%s in field value", err))
					}
					panic(tc.errorf(node, "%s", err))
				}
				if !isExported(fieldTi.Name) {
					panic(tc.errorf(node, "implicit assignment of unexported field '%s' in %v", fieldTi.Name, node))
				}
				keyValue.Key = ast.NewIdentifier(nil, fieldTi.Name)
				valueTi.SetValue(fieldTi.Type)
			}
		}

	case reflect.Array:

		hasIndex := map[int]struct{}{}
		for i := range node.KeyValues {
			kv := &node.KeyValues[i]
			if kv.Key != nil {
				keyTi := tc.checkExpression(kv.Key)
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
				elemTi = tc.checkExpression(kv.Value)
			}
			if err := isAssignableTo(elemTi, kv.Value, ti.Type.Elem()); err != nil {
				k := ti.Type.Elem().Kind()
				if _, ok := err.(invalidTypeInAssignment); ok && k == reflect.Slice || k == reflect.Array {
					panic(tc.errorf(node, "%s in array or slice literal", err))
				}
				panic(tc.errorf(node, "%s", err))
			}
			elemTi.SetValue(ti.Type.Elem())
		}

	case reflect.Slice:

		hasIndex := map[int]struct{}{}
		for i := range node.KeyValues {
			kv := &node.KeyValues[i]
			if kv.Key != nil {
				keyTi := tc.checkExpression(kv.Key)
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
				elemTi = tc.checkExpression(kv.Value)
			}
			if err := isAssignableTo(elemTi, kv.Value, ti.Type.Elem()); err != nil {
				k := ti.Type.Elem().Kind()
				if _, ok := err.(invalidTypeInAssignment); ok {
					if k == reflect.Slice || k == reflect.Array {
						panic(tc.errorf(node, "%s in array or slice literal", err))
					}
					panic(tc.errorf(node, "cannot convert %s (type %s) to type %v", kv.Value, elemTi, ti.Type.Elem()))
				}
				panic(tc.errorf(node, "%s", err))
			}
			elemTi.SetValue(ti.Type.Elem())
		}

	case reflect.Map:

		hasKey := map[interface{}]struct{}{}
		for i := range node.KeyValues {
			kv := &node.KeyValues[i]
			keyType := ti.Type.Key()
			elemType := ti.Type.Elem()
			var keyTi *TypeInfo
			if compLit, ok := kv.Key.(*ast.CompositeLiteral); ok {
				keyTi = tc.checkCompositeLiteral(compLit, keyType)
			} else {
				keyTi = tc.checkExpression(kv.Key)
			}
			if err := isAssignableTo(keyTi, kv.Key, keyType); err != nil {
				if _, ok := err.(invalidTypeInAssignment); ok {
					panic(tc.errorf(node, "%s in map key", err))
				}
				panic(tc.errorf(node, "%s", err))
			}
			if keyTi.IsConstant() {
				key := typedValue(keyTi, keyType)
				if _, ok := hasKey[key]; ok {
					panic(tc.errorf(node, "duplicate key %s in map literal", kv.Key))
				}
				hasKey[key] = struct{}{}
			}
			keyTi.SetValue(keyType)
			var valueTi *TypeInfo
			if cl, ok := kv.Value.(*ast.CompositeLiteral); ok {
				valueTi = tc.checkCompositeLiteral(cl, elemType)
			} else {
				valueTi = tc.checkExpression(kv.Value)
			}
			if err := isAssignableTo(valueTi, kv.Value, elemType); err != nil {
				if _, ok := err.(invalidTypeInAssignment); ok {
					panic(tc.errorf(node, "%s in map value", err))
				}
				panic(tc.errorf(node, "%s", err))
			}
			valueTi.SetValue(elemType)
		}

	}

	nodeTi := &TypeInfo{Type: ti.Type}
	tc.TypeInfo[node] = nodeTi

	return nodeTi
}
