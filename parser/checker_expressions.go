// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strings"
	"unicode"

	"scrigo/ast"
)

var errDivisionByZero = errors.New("division by zero")

const noEllipses = -1

const (
	maxInt  = int(maxUint >> 1)
	minInt  = -maxInt - 1
	maxUint = ^uint(0)
)

var integerRanges = [...]struct{ min, max *big.Int }{
	{big.NewInt(int64(minInt)), big.NewInt(int64(maxInt))}, // int
	{big.NewInt(-1 << 7), big.NewInt(1<<7 - 1)},            // int8
	{big.NewInt(-1 << 15), big.NewInt(1<<15 - 1)},          // int16
	{big.NewInt(-1 << 31), big.NewInt(1<<31 - 1)},          // int32
	{big.NewInt(-1 << 63), big.NewInt(1<<63 - 1)},          // int64
	{nil, big.NewInt(0).SetUint64(uint64(maxUint))},        // uint
	{nil, big.NewInt(1<<8 - 1)},                            // uint8
	{nil, big.NewInt(1<<16 - 1)},                           // uint16
	{nil, big.NewInt(1<<32 - 1)},                           // uint32
	{nil, big.NewInt(0).SetUint64(1<<64 - 1)},              // uint64
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
	reflect.Complex64:     true,
	reflect.Complex128:    true,
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
var stringType = reflect.TypeOf("")
var intType = reflect.TypeOf(0)
var uint8Type = reflect.TypeOf(uint8(0))
var int32Type = reflect.TypeOf(int32(0))
var float64Type = reflect.TypeOf(float64(0))

var builtinTypeInfo = &ast.TypeInfo{Properties: ast.PropertyIsBuiltin}
var uint8TypeInfo = &ast.TypeInfo{Type: uint8Type, Properties: ast.PropertyIsType}
var int32TypeInfo = &ast.TypeInfo{Type: int32Type, Properties: ast.PropertyIsType}

var untypedBoolTypeInfo = &ast.TypeInfo{Type: boolType, Properties: ast.PropertyUntyped}

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
	"nil":        &ast.TypeInfo{Properties: ast.PropertyNil},
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
	"float64":    &ast.TypeInfo{Type: float64Type, Properties: ast.PropertyIsType},
	"false":      &ast.TypeInfo{Type: boolType, Properties: ast.PropertyIsConstant | ast.PropertyUntyped, Value: false},
	"int":        &ast.TypeInfo{Type: intType, Properties: ast.PropertyIsType},
	"int16":      &ast.TypeInfo{Type: reflect.TypeOf(int16(0)), Properties: ast.PropertyIsType},
	"int32":      int32TypeInfo,
	"int64":      &ast.TypeInfo{Type: reflect.TypeOf(int64(0)), Properties: ast.PropertyIsType},
	"int8":       &ast.TypeInfo{Type: reflect.TypeOf(int8(0)), Properties: ast.PropertyIsType},
	"rune":       int32TypeInfo,
	"string":     &ast.TypeInfo{Type: stringType, Properties: ast.PropertyIsType},
	"true":       &ast.TypeInfo{Type: boolType, Properties: ast.PropertyIsConstant | ast.PropertyUntyped, Value: true},
	"uint":       &ast.TypeInfo{Type: reflect.TypeOf(uint(0)), Properties: ast.PropertyIsType},
	"uint16":     &ast.TypeInfo{Type: reflect.TypeOf(uint32(0)), Properties: ast.PropertyIsType},
	"uint32":     &ast.TypeInfo{Type: reflect.TypeOf(uint32(0)), Properties: ast.PropertyIsType},
	"uint64":     &ast.TypeInfo{Type: reflect.TypeOf(uint64(0)), Properties: ast.PropertyIsType},
	"uint8":      uint8TypeInfo,
	"uintptr":    &ast.TypeInfo{Type: reflect.TypeOf(uintptr(0)), Properties: ast.PropertyIsType},
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
	Ident           string          // identifier of the declaration.
	Type            ast.Expression  // nil if declaration has no type.
	DeclarationType DeclarationType // constant, variable or function.
	Value           ast.Node        // ast.Expression for variables/constant, ast.Block for functions.
}

// typechecker represents the state of a type checking.
// TODO (Gianluca): join fileBlock and packageBlock, making them a single block.
type typechecker struct {
	path         string
	imports      map[string]packageInfo // TODO (Gianluca): review!
	fileBlock    typeCheckerScope
	packageBlock typeCheckerScope
	scopes       []typeCheckerScope
	ancestors    []*ancestor
	terminating  bool // https://golang.org/ref/spec#Terminating_statements
	hasBreak     map[ast.Node]bool
	context      ast.Context

	// Variable initialization support structures.
	// TODO (Gianluca): can be simplified?
	declarations        []*Declaration      // global declarations.
	initOrder           []string            // global variables initialization order.
	varDeps             map[string][]string // key is a variable, value is list of its dependencies.
	currentIdent        string              // identifier currently being evaluated.
	currentlyEvaluating []string            // stack of identifiers used in a single evaluation.
	temporaryEvaluated  map[string]*ast.TypeInfo
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

// AddScope adds a new empty scope to the type checker.
func (tc *typechecker) AddScope() {
	tc.scopes = append(tc.scopes, make(typeCheckerScope))
}

// RemoveCurrentScope removes the current scope from the type checker.
func (tc *typechecker) RemoveCurrentScope() {
	tc.scopes = tc.scopes[:len(tc.scopes)-1]
}

// LookupScopes looks up name in the scopes. Returns the type info of the name or
// false if the name does not exist. If justCurrentScope is true, LookupScopes
// looks up only in the current scope.
func (tc *typechecker) LookupScopes(name string, justCurrentScope bool) (*ast.TypeInfo, bool) {
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
	for n, ti := range tc.packageBlock {
		if n == name {
			return ti, true
		}
	}
	for n, ti := range tc.fileBlock {
		if n == name {
			return ti, true
		}
	}
	return nil, false
}

// AssignScope assigns value to name in the last scope.
func (tc *typechecker) AssignScope(name string, value *ast.TypeInfo) {
	tc.scopes[len(tc.scopes)-1][name] = value
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

func (tc *typechecker) CheckUpValue(name string) string {
	_, funcBound := tc.getCurrentFunc()
	for i := len(tc.scopes) - 1; i >= 0; i-- {
		for n, _ := range tc.scopes[i] {
			if n != name {
				continue
			}
			if i < funcBound-1 { // out of current function scope.
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

	// Upvalues.
	if fun, _ := tc.getCurrentFunc(); fun != nil {
		uv := tc.CheckUpValue(ident.Name)
		if uv != "" {
			fun.Upvalues = append(fun.Upvalues, uv)
		}
	}

	i, ok := tc.LookupScopes(ident.Name, false)
	if !ok {
		panic(tc.errorf(ident, "undefined: %s", ident.Name))
	}

	if tmpTi, ok := tc.temporaryEvaluated[ident.Name]; ok {
		return tmpTi
	}

	if tc.getDecl(ident.Name) != nil {
		tc.varDeps[tc.currentIdent] = append(tc.varDeps[tc.currentIdent], ident.Name)
		tc.currentlyEvaluating = append(tc.currentlyEvaluating, ident.Name)
		if containsDuplicates(tc.currentlyEvaluating) {
			// TODO (Gianluca): add positions.
			panic(tc.errorf(ident, "initialization loop:\n\t%s", strings.Join(tc.currentlyEvaluating, " refers to\n\t")))
		}
	}

	// Check bodies of global functions.
	// TODO (Gianluca): this must be done only when checking global variables.
	if i.Type != nil && i.Type.Kind() == reflect.Func && !i.Addressable() {
		tc.checkNodes(tc.getDecl(ident.Name).Value.(*ast.Block).Nodes)
	}

	// Global declaration.
	if i == notChecked {
		switch d := tc.getDecl(ident.Name); d.DeclarationType {
		case DeclarationConstant, DeclarationVariable:
			ti := tc.checkExpression(d.Value.(ast.Expression))
			tc.temporaryEvaluated[ident.Name] = ti
			return ti
		case DeclarationFunction:
			tc.checkNodes(d.Value.(*ast.Block).Nodes)
			return &ast.TypeInfo{Type: tc.typeof(d.Type, noEllipses).Type}
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
			Type:       stringType,
			Properties: ast.PropertyUntyped | ast.PropertyIsConstant,
			Value:      expr.Text,
		}

	case *ast.Int:
		return &ast.TypeInfo{
			Type:       intType,
			Properties: ast.PropertyUntyped | ast.PropertyIsConstant,
			Value:      &expr.Value,
		}

	case *ast.Rune:
		return &ast.TypeInfo{
			Type:       int32Type,
			Properties: ast.PropertyUntyped | ast.PropertyIsConstant,
			Value:      big.NewInt(int64(expr.Value)),
		}

	case *ast.Float:
		return &ast.TypeInfo{
			Type:       float64Type,
			Properties: ast.PropertyUntyped | ast.PropertyIsConstant,
			Value:      &expr.Value,
		}

	case *ast.Parentesis:
		panic("unexpected parentesis")

	case *ast.UnaryOperator:
		_ = tc.checkExpression(expr.Expr)
		t, err := tc.unaryOp(expr)
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
		if !len.IsConstant() {
			panic(tc.errorf(expr, "non-constant array bound %s", expr.Len))
		}
		declLength, err := tc.convert(len, intType, false)
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
		return &ast.TypeInfo{Properties: ast.PropertyIsType, Type: reflect.ArrayOf(length, elem.Type)}

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
			if res == nil {
				out[i] = out[i+1]
			} else {
				c := tc.checkType(res.Type, noEllipses)
				out[i] = c.Type
			}
		}
		return &ast.TypeInfo{Type: reflect.FuncOf(in, out, variadic), Properties: ast.PropertyIsType}

	case *ast.Func:
		tc.AddScope()
		t := tc.checkType(expr.Type, noEllipses)
		tc.ancestors = append(tc.ancestors, &ancestor{len(tc.scopes), expr})
		// Adds parameters to the function body scope.
		for _, f := range expr.Type.Parameters {
			if f.Ident != nil {
				t := tc.checkType(f.Type, noEllipses)
				tc.AssignScope(f.Ident.Name, &ast.TypeInfo{Type: t.Type, Properties: ast.PropertyAddressable})
			}
		}
		// Adds named return values to the function body scope.
		for _, f := range expr.Type.Result {
			if f.Ident != nil {
				t := tc.checkType(f.Type, noEllipses)
				tc.AssignScope(f.Ident.Name, &ast.TypeInfo{Type: t.Type, Properties: ast.PropertyAddressable})
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
		return types[0]

	case *ast.Index:
		t := tc.checkExpression(expr.Expr)
		if t.Nil() {
			panic(tc.errorf(expr, "use of untyped nil"))
		}
		kind := t.Type.Kind()
		switch kind {
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
					err = fmt.Errorf("non-integer %s index %s", k, expr.Index)
				}
				panic(tc.errorf(expr.Index, "%s", err))
			}
			if v != nil {
				n := int(v.(*big.Int).Int64())
				if n < 0 {
					panic(tc.errorf(expr, "invalid %s index %s (index must be non-negative)", k, expr.Index))
				}
				if t.Value != nil {
					if s := t.Value.(string); n >= len(s) {
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
		var lv, hv int
		if expr.Low != nil {
			low := tc.checkExpression(expr.Low)
			v, err := tc.convert(low, intType, false)
			if err != nil {
				if err == errTypeConversion {
					err = fmt.Errorf("invalid slice index %s (type %s)", expr.Low, low)
				}
				panic(tc.errorf(expr, "%s", err))
			}
			if v != nil {
				lv = int(v.(*big.Int).Int64())
				if lv < 0 {
					panic(tc.errorf(expr, "invalid slice index %s (index must be non-negative)", expr.Low))
				}
				if t.Value != nil {
					if s := t.Value.(string); lv > len(s) {
						panic(tc.errorf(expr, "invalid slice index %d (out of bounds for %d-byte string)", lv, len(s)))
					}
				}
			}
		}
		if expr.High != nil {
			high := tc.checkExpression(expr.High)
			v, err := tc.convert(high, intType, false)
			if err != nil {
				if err == errTypeConversion {
					err = fmt.Errorf("invalid slice index %s (type %s)", expr.High, high)
				}
				panic(tc.errorf(expr, "%s", err))
			}
			if v != nil {
				hv = int(v.(*big.Int).Int64())
				if hv < 0 {
					panic(tc.errorf(expr, "invalid slice index %s (index must be non-negative)", expr.High))
				}
				if t.Value != nil {
					if s := t.Value.(string); hv > len(s) {
						panic(tc.errorf(expr, "invalid slice index %d (out of bounds for %d-byte string)", hv, len(s)))
					}
				}
				if lv > hv {
					panic(tc.errorf(expr, "invalid slice index: %d > %d", lv, hv))
				}
			}
		}
		if t.Type == nil {
			return &ast.TypeInfo{Type: universe["string"].Type}
		}
		switch kind {
		case reflect.String, reflect.Slice:
			return &ast.TypeInfo{Type: t.Type}
		case reflect.Array:
			return &ast.TypeInfo{Type: reflect.SliceOf(t.Type.Elem())}
		case reflect.Ptr:
			return &ast.TypeInfo{Type: reflect.SliceOf(t.Type.Elem().Elem())}
		}

	case *ast.Selector:
		// Package selector.
		if ident, ok := expr.Expr.(*ast.Identifier); ok {
			ti, ok := tc.LookupScopes(ident.Name, false)
			if ok {
				if ti.IsPackage() {
					if !unicode.Is(unicode.Lu, []rune(expr.Ident)[0]) {
						panic(tc.errorf(expr, "cannot refer to unexported name %s", expr))
					}
					pkg := ti.Value.(*packageInfo)
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

// unaryOp executes an unary expression and returns its result. Returns an
// error if the operation can not be executed.
func (tc *typechecker) unaryOp(expr *ast.UnaryOperator) (*ast.TypeInfo, error) {

	t := expr.Expr.TypeInfo()
	k := t.Type.Kind()

	ti := &ast.TypeInfo{
		Type:       t.Type,
		Properties: t.Properties & (ast.PropertyUntyped | ast.PropertyIsConstant),
	}

	switch expr.Op {
	case ast.OperatorNot:
		if t.Nil() || k != reflect.Bool {
			return nil, fmt.Errorf("invalid operation: ! %s", t)
		}
		if t.IsConstant() {
			ti.Value = !t.Value.(bool)
		}
	case ast.OperatorAddition:
		if t.Nil() || !numericKind[k] {
			return nil, fmt.Errorf("invalid operation: + %s", t)
		}
		if t.IsConstant() {
			ti.Value = t.Value
		}
	case ast.OperatorSubtraction:
		if t.Nil() || !numericKind[k] {
			return nil, fmt.Errorf("invalid operation: - %s", t)
		}
		if t.IsConstant() {
			if t.Untyped() {
				switch v := t.Value.(type) {
				case *big.Int:
					ti.Value = (&big.Int{}).Neg(v)
				case *big.Float:
					v = (&big.Float{}).Neg(v)
				case *big.Rat:
					v = (&big.Rat{}).Neg(v)
				}
			} else {
				switch v := t.Value.(type) {
				case *big.Int:
					v = (&big.Int{}).Neg(v)
					min := integerRanges[k-2].min
					max := integerRanges[k-2].max
					if min == nil && v.Sign() < 0 || min != nil && v.Cmp(min) < 0 || v.Cmp(max) > 0 {
						return nil, fmt.Errorf("constant %v overflows %s", v, t)
					}
					ti.Value = v
				case *big.Float:
					v = (&big.Float{}).Neg(v)
					var acc big.Accuracy
					if k == reflect.Float64 {
						_, acc = v.Float64()
					} else {
						_, acc = v.Float32()
					}
					if acc != 0 {
						return nil, fmt.Errorf("constant %v overflows %s", v, t)
					}
					ti.Value = v
				}
			}
		}
	case ast.OperatorMultiplication:
		if t.Type.Kind() != reflect.Ptr {
			return nil, fmt.Errorf("invalid indirect of %s (type %s)", expr, t)
		}
		ti.Type = t.Type.Elem()
		ti.Properties = ti.Properties | ast.PropertyAddressable
	case ast.OperatorAmpersand:
		if _, ok := expr.Expr.(*ast.CompositeLiteral); !ok && !t.Addressable() {
			return nil, fmt.Errorf("cannot take the address of %s", expr)
		}
		ti.Type = reflect.PtrTo(t.Type)
	}

	return ti, nil
}

// isComparison reports whether op is a comparison operator.
func isComparison(op ast.OperatorType) bool {
	return op >= ast.OperatorEqual && op <= ast.OperatorGreaterOrEqual
}

// binaryOp executes the binary expression t1 op t2 and returns its result.
// Returns an error if the operation can not be executed.
func (tc *typechecker) binaryOp(expr *ast.BinaryOperator) (*ast.TypeInfo, error) {

	t1 := tc.checkExpression(expr.Expr1)
	t2 := tc.checkExpression(expr.Expr2)

	if t1.Untyped() && t2.Untyped() {
		return tc.uBinaryOp(t1, expr, t2)
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
			return nil, fmt.Errorf("invalid operation: %v (operator %s not defined on %s", expr, op, k)
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
		v, err := tc.convert(t1, t2.Type, false)
		if err != nil {
			if err == errTypeConversion {
				return nil, fmt.Errorf("cannot convert %v (type %s) to type %s", expr, t1, t2)
			}
			panic(tc.errorf(expr, "%s", err))
		}
		t1 = &ast.TypeInfo{Type: t2.Type, Properties: ast.PropertyIsConstant, Value: v}
	} else if t2.Untyped() {
		v, err := tc.convert(t2, t1.Type, false)
		if err != nil {
			if err == errTypeConversion {
				panic(tc.errorf(expr, "cannot convert %v (type %s) to type %s", expr, t2, t1))
			}
			panic(tc.errorf(expr, "%s", err))
		}
		t2 = &ast.TypeInfo{Type: t1.Type, Properties: ast.PropertyIsConstant, Value: v}
	}

	if t1.IsConstant() && t2.IsConstant() {
		return tc.tBinaryOp(t1, expr, t2)
	}

	if isComparison(expr.Op) {
		if !tc.isAssignableTo(t1, t2.Type) && !tc.isAssignableTo(t2, t1.Type) {
			panic(tc.errorf(expr, "invalid operation: %v (mismatched types %s and %s)", expr, t1.ShortString(), t2.ShortString()))
		}
		if expr.Op == ast.OperatorEqual || expr.Op == ast.OperatorNotEqual {
			if !t1.Type.Comparable() {
				// TODO(marco) explain in the error message why they are not comparable.
				panic(tc.errorf(expr, "invalid operation: %v (%s cannot be compared)", expr, t1.Type))
			}
		} else if !tc.isOrdered(t1) {
			panic(tc.errorf(expr, "invalid operation: %v (operator %s not defined on %s)", expr, expr.Op, t1.Type.Kind()))
		}
		return &ast.TypeInfo{Type: boolType, Properties: ast.PropertyUntyped}, nil
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

// tBinaryOp executes a binary expression where the operands are typed
// constants and returns its result. Returns an error if the operation can not
// be executed.
func (tc *typechecker) tBinaryOp(t1 *ast.TypeInfo, expr *ast.BinaryOperator, t2 *ast.TypeInfo) (*ast.TypeInfo, error) {

	if t1.Type != t2.Type {
		return nil, fmt.Errorf("invalid operation: %v (mismatched types %s and %s)", expr, t1, t2)
	}

	t := &ast.TypeInfo{Type: boolType, Properties: ast.PropertyIsConstant}

	switch v1 := t1.Value.(type) {
	case bool:
		v2 := t2.Value.(bool)
		switch expr.Op {
		case ast.OperatorEqual:
			t.Value = v1 == v2
		case ast.OperatorNotEqual:
			t.Value = v1 != v2
		case ast.OperatorAnd:
			t.Value = v1 && v2
		case ast.OperatorOr:
			t.Value = v1 || v2
		default:
			return nil, fmt.Errorf("invalid operation: %v (operator %s not defined on %s)", expr, expr.Op, t1)
		}
	case string:
		v2 := t2.Value.(string)
		switch expr.Op {
		case ast.OperatorEqual:
			t.Value = v1 == v2
		case ast.OperatorNotEqual:
			t.Value = v1 != v2
		case ast.OperatorAddition:
			t.Value = v1 + v2
			t.Type = t1.Type
		default:
			return nil, fmt.Errorf("invalid operation: %v (operator %s not defined on %s)", expr, expr.Op, t1)
		}
	case *big.Int:
		v2 := t2.Value.(*big.Int)
		switch expr.Op {
		case ast.OperatorEqual:
			t.Value = v1.Cmp(v2) == 0
		case ast.OperatorNotEqual:
			t.Value = v1.Cmp(v2) != 0
		case ast.OperatorLess:
			t.Value = v1.Cmp(v2) < 0
		case ast.OperatorLessOrEqual:
			t.Value = v1.Cmp(v2) <= 0
		case ast.OperatorGreater:
			t.Value = v1.Cmp(v2) > 0
		case ast.OperatorGreaterOrEqual:
			t.Value = v1.Cmp(v2) >= 0
		case ast.OperatorAddition:
			t.Value = big.NewInt(0).Add(v1, v2)
		case ast.OperatorSubtraction:
			t.Value = big.NewInt(0).Sub(v1, v2)
		case ast.OperatorMultiplication:
			t.Value = big.NewInt(0).Mul(v1, v2)
		case ast.OperatorDivision:
			t.Value = big.NewInt(0).Quo(v1, v2)
		case ast.OperatorModulo:
			t.Value = big.NewInt(0).Rem(v1, v2)
		}
		if v, ok := t.Value.(*big.Int); ok {
			min := integerRanges[t1.Type.Kind()-2].min
			max := integerRanges[t1.Type.Kind()-2].max
			if min == nil && v.Sign() < 0 || min != nil && v.Cmp(min) < 0 || v.Cmp(max) > 0 {
				return nil, fmt.Errorf("constant %v overflows %s", v, t1)
			}
			t.Type = t1.Type
		}
	case *big.Float:
		v2 := t2.Value.(*big.Float)
		switch expr.Op {
		case ast.OperatorEqual:
			t.Value = v1.Cmp(v2) == 0
		case ast.OperatorNotEqual:
			t.Value = v1.Cmp(v2) != 0
		case ast.OperatorLess:
			t.Value = v1.Cmp(v2) < 0
		case ast.OperatorLessOrEqual:
			t.Value = v1.Cmp(v2) <= 0
		case ast.OperatorGreater:
			t.Value = v1.Cmp(v2) > 0
		case ast.OperatorGreaterOrEqual:
			t.Value = v1.Cmp(v2) >= 0
		case ast.OperatorAddition:
			t.Value = big.NewFloat(0).Add(v1, v2)
		case ast.OperatorSubtraction:
			t.Value = big.NewFloat(0).Sub(v1, v2)
		case ast.OperatorMultiplication:
			t.Value = big.NewFloat(0).Mul(v1, v2)
		case ast.OperatorDivision:
			t.Value = big.NewFloat(0).Quo(v1, v2)
		case ast.OperatorModulo:
			return nil, fmt.Errorf("invalid operation: %v (operator %% not defined on %s)", expr, t1)
		}
		if v, ok := t.Value.(*big.Float); ok {
			var acc big.Accuracy
			if t1.Type.Kind() == reflect.Float64 {
				_, acc = v.Float64()
			} else {
				_, acc = v.Float32()
			}
			if acc != 0 {
				return nil, fmt.Errorf("constant %v overflows %s", v, t1)
			}
			t.Type = t1.Type
		}
	}

	return t, nil
}

// uBinaryOp executes a binary expression where the operands are untyped and
// returns its result. Returns an error if the operation can not be executed.
func (tc *typechecker) uBinaryOp(t1 *ast.TypeInfo, expr *ast.BinaryOperator, t2 *ast.TypeInfo) (*ast.TypeInfo, error) {

	k1 := t1.Type.Kind()
	k2 := t2.Type.Kind()

	if !(k1 == k2 || numericKind[k1] && numericKind[k2]) {
		return nil, fmt.Errorf("invalid operation: %v (mismatched types %s and %s)",
			expr, t1.ShortString(), t2.ShortString())
	}

	t := &ast.TypeInfo{Type: boolType, Properties: ast.PropertyUntyped | ast.PropertyIsConstant}

	switch k1 {
	case reflect.Bool:
		switch expr.Op {
		case ast.OperatorEqual:
			t.Value = t1.Value.(bool) == t2.Value.(bool)
		case ast.OperatorNotEqual:
			t.Value = t1.Value.(bool) != t2.Value.(bool)
		case ast.OperatorAnd:
			t.Value = t1.Value.(bool) && t2.Value.(bool)
		case ast.OperatorOr:
			t.Value = t1.Value.(bool) || t2.Value.(bool)
		default:
			return nil, fmt.Errorf("invalid operation: %v (operator %s not defined on %s)", expr, expr.Op, t1.ShortString())
		}
	case reflect.String:
		switch expr.Op {
		case ast.OperatorEqual:
			t.Value = t1.Value.(string) == t2.Value.(string)
		case ast.OperatorNotEqual:
			t.Value = t1.Value.(string) != t2.Value.(string)
		case ast.OperatorAddition:
			t.Value = t1.Value.(string) + t2.Value.(string)
			t.Type = stringType
		default:
			return nil, fmt.Errorf("invalid operation: %v (operator %s not defined on %s)", expr, expr.Op, t1.ShortString())
		}
	default:
		v1, v2 := toSameType(t1.Value, t2.Value)
		switch expr.Op {
		default:
			var cmp int
			switch v1 := v1.(type) {
			case *big.Int:
				cmp = v1.Cmp(v2.(*big.Int))
			case *big.Float:
				cmp = v1.Cmp(v2.(*big.Float))
			case *big.Rat:
				cmp = v1.Cmp(v2.(*big.Rat))
			}
			switch expr.Op {
			case ast.OperatorEqual:
				t.Value = cmp == 0
			case ast.OperatorNotEqual:
				t.Value = cmp != 0
			case ast.OperatorLess:
				t.Value = cmp < 0
			case ast.OperatorLessOrEqual:
				t.Value = cmp <= 0
			case ast.OperatorGreater:
				t.Value = cmp > 0
			case ast.OperatorGreaterOrEqual:
				t.Value = cmp >= 0
			default:
				return nil, fmt.Errorf("invalid operation: %v (operator %s not defined on untyped number)", expr, expr.Op)
			}
			return t, nil
		case ast.OperatorAddition:
			switch v2 := v2.(type) {
			case *big.Int:
				t.Value = (&big.Int{}).Add(v1.(*big.Int), v2)
			case *big.Float:
				t.Value = (&big.Float{}).Add(v1.(*big.Float), v2)
			case *big.Rat:
				t.Value = (&big.Rat{}).Add(v1.(*big.Rat), v2)
			}
		case ast.OperatorSubtraction:
			switch v2 := t2.Value.(type) {
			case *big.Int:
				t.Value = (&big.Int{}).Sub(v1.(*big.Int), v2)
			case *big.Float:
				t.Value = (&big.Float{}).Sub(v1.(*big.Float), v2)
			case *big.Rat:
				t.Value = (&big.Rat{}).Sub(v1.(*big.Rat), v2)
			}
		case ast.OperatorMultiplication:
			switch v2 := v2.(type) {
			case *big.Int:
				t.Value = (&big.Int{}).Sub(v1.(*big.Int), v2)
			case *big.Float:
				t.Value = (&big.Float{}).Sub(v1.(*big.Float), v2)
			case *big.Rat:
				t.Value = (&big.Rat{}).Sub(v1.(*big.Rat), v2)
			}
		case ast.OperatorDivision:
			switch v2 := t2.Value.(type) {
			case *big.Int:
				if v2.Sign() == 0 {
					return nil, errDivisionByZero
				}
				t.Value = (&big.Rat{}).SetFrac(v1.(*big.Int), v2)
			case *big.Float:
				if v2.Sign() == 0 {
					return nil, errDivisionByZero
				}
				t.Value = (&big.Float{}).Quo(v1.(*big.Float), v2)
			case *big.Rat:
				if v2.Sign() == 0 {
					return nil, errDivisionByZero
				}
				t.Value = (&big.Rat{}).Quo(v1.(*big.Rat), v2)
			}
		case ast.OperatorModulo:
			if k1 == reflect.Float64 || k2 == reflect.Float64 {
				return nil, errors.New("illegal constant expression: floating-point % operation")
			}
			vv2 := v2.(*big.Int)
			if vv2.Sign() == 0 {
				return nil, errDivisionByZero
			}
			t.Value = (&big.Rat{}).SetFrac(v1.(*big.Int), vv2)
		}
		t.Type = t1.Type
		if k2 > k1 {
			t.Type = t2.Type
		}
	}

	return t, nil
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
		ti := &ast.TypeInfo{Type: t.Type, Value: value}
		if value != nil {
			ti.Properties = ast.PropertyIsConstant
		}
		return []*ast.TypeInfo{ti}
	}

	if t.IsPackage() {
		// TODO (Gianluca): why "fmt"?
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
								if lenvalue = int(c.(*big.Int).Int64()); lenvalue < 0 {
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
								if capvalue = int(v.(*big.Int).Int64()); capvalue < 0 {
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
							if ok && int(v.(*big.Int).Int64()) < 0 {
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

		case "println":
			// TODO
			return nil

		case "print":
			// TODO
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
				have += c.ShortString()
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
			panic(tc.errorf(expr.Args[i], "cannot use %s (type %s) as type %s in argument to %s", expr.Args[i], a.ShortString(), in, expr.Func))
		}
	}

	numOut := t.Type.NumOut()
	resultTypes := make([]*ast.TypeInfo, numOut)
	for i := 0; i < numOut; i++ {
		resultTypes[i] = &ast.TypeInfo{Type: t.Type.Out(i)}
	}

	return resultTypes
}

var errTypeConversion = errors.New("failed type conversion")

// convert converts a value. explicit reports whether the conversion is
// explicit. If the converted value is a constant, convert returns its value,
// otherwise returns nil.
//
// If the value can not be converted, returns an errTypeConversion type error,
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

	if ti.IsConstant() && k2 != reflect.Interface {
		if explicit {
			k1 := t.Kind()
			if k2 == reflect.String && reflect.Int <= k1 && k1 <= reflect.Uint64 {
				// As a special case, an integer constant can be explicitly
				// converted to a string type.
				switch v := v.(type) {
				case *big.Int:
					if v.IsInt64() {
						return string(v.Int64()), nil
					}
				}
				return "\uFFFD", nil
			} else if k2 == reflect.Slice && k1 == reflect.String {
				// As a special case, a string constant can be explicitly converted
				// to a slice of runes or bytes.
				if elem := t2.Elem(); elem == uint8Type || elem == int32Type {
					return nil, nil
				}
			}
		}
		return tc.representedBy(ti, t2)
	}

	if t.ConvertibleTo(t2) {
		return nil, nil
	}

	return nil, errTypeConversion
}

// representedBy returns a constant value represented as a value of type t2.
func (tc *typechecker) representedBy(t1 *ast.TypeInfo, t2 reflect.Type) (interface{}, error) {

	v := t1.Value
	k := t2.Kind()

	switch v := v.(type) {
	case bool:
		if k == reflect.Bool {
			return v, nil
		}
	case string:
		if k == reflect.String {
			return v, nil
		}
	case *big.Int:
		switch {
		case reflect.Int <= k && k <= reflect.Uint64:
			if t1.Untyped() || k != t1.Type.Kind() {
				min := integerRanges[k-2].min
				max := integerRanges[k-2].max
				if min == nil && v.Sign() < 0 || min != nil && v.Cmp(min) < 0 || v.Cmp(max) > 0 {
					return nil, fmt.Errorf("constant %v overflows %s", v, t2)
				}
			}
			return v, nil
		case k == reflect.Float64:
			n := (&big.Float{}).SetInt(v)
			if t1.Untyped() && !v.IsInt64() && !v.IsUint64() {
				if _, acc := n.Float64(); acc != big.Exact {
					return nil, fmt.Errorf("constant %v overflows %s", v, t2)
				}
			}
			return n, nil
		case k == reflect.Float32:
			n := (&big.Float{}).SetInt(v).SetPrec(24)
			if t1.Untyped() && !v.IsInt64() && !v.IsUint64() {
				if _, acc := n.Float32(); acc != big.Exact {
					return nil, fmt.Errorf("constant %v overflows %s", v, t2)
				}
			}
			return n, nil
		}
	case *big.Float:
		switch {
		case reflect.Int <= k && k <= reflect.Uint64:
			if n, acc := v.Int(nil); acc == big.Exact {
				min := integerRanges[k-2].min
				max := integerRanges[k-2].max
				if (min != nil || v.Sign() >= 0) && (min == nil || n.Cmp(min) >= 0) && n.Cmp(max) <= 0 {
					return n, nil
				}
			}
			return nil, fmt.Errorf("constant %v truncated to integer", v)
		case k == reflect.Float64:
			if t1.Untyped() {
				n, _ := v.Float64()
				if math.IsInf(n, 0) {
					return nil, fmt.Errorf("constant %v overflows %s", v, t2)
				}
				v = big.NewFloat(n)
			}
			return v, nil
		case k == reflect.Float32:
			n, _ := v.Float32()
			if math.IsInf(float64(n), 0) {
				return nil, fmt.Errorf("constant %v overflows %s", v, t2)
			}
			return big.NewFloat(float64(n)), nil
		}
	case *big.Rat:
		switch {
		case reflect.Int <= k && k <= reflect.Uint64:
			if v.IsInt() {
				n := v.Num()
				min := integerRanges[k-2].min
				max := integerRanges[k-2].max
				if (min != nil || v.Sign() >= 0) && (min == nil || n.Cmp(min) >= 0) && n.Cmp(max) <= 0 {
					return n, nil
				}
			}
			return nil, fmt.Errorf("constant %v truncated to integer", v)
		case k == reflect.Float64:
			n, _ := v.Float64()
			if math.IsInf(n, 0) {
				return nil, fmt.Errorf("constant %v overflows %s", v, t2)
			}
			return big.NewFloat(n), nil
		case k == reflect.Float32:
			n, _ := v.Float32()
			if math.IsInf(float64(n), 0) {
				return nil, fmt.Errorf("constant %v overflows %s", v, t2)
			}
			return big.NewFloat(float64(n)), nil
		}
	}

	return nil, fmt.Errorf("cannot convert %v (type %s) to type %s", v, t1, t2)
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

// isOrdered reports whether t is ordered.
func (tc *typechecker) isOrdered(t *ast.TypeInfo) bool {
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

// toSameType returns v1 and v2 with the same types.
func toSameType(v1, v2 interface{}) (interface{}, interface{}) {
	switch n1 := v1.(type) {
	case *big.Int:
		switch v2.(type) {
		case *big.Float:
			v1 = (&big.Float{}).SetInt(n1)
		case *big.Rat:
			v1 = (&big.Rat{}).SetInt(n1)
		}
	case *big.Float:
		switch n2 := v2.(type) {
		case *big.Int:
			v2 = (&big.Float{}).SetInt(n2)
		case *big.Rat:
			num := (&big.Float{}).SetInt(n2.Num())
			den := (&big.Float{}).SetInt(n2.Denom())
			v2 = num.Quo(num, den)
		}
	case *big.Rat:
		switch n2 := v2.(type) {
		case *big.Int:
			v2 = (&big.Rat{}).SetInt(n2)
		case *big.Float:
			num := (&big.Float{}).SetInt(n1.Num())
			den := (&big.Float{}).SetInt(n1.Denom())
			v1 = num.Quo(num, den)
		}
	}
	return v1, v2
}
