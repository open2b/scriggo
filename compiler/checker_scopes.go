// Copyright (c) 2021 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"

	"github.com/open2b/scriggo/compiler/ast"
)

// scopes represents the universe block, global block, file/package block and
// function scopes.
//
//   0, 1 are the universe block. 1 is for the format types.
//   2    is the global block, empty for programs.
//   3    is the file/package block.
//   4+   are the scopes in functions. For scripts, 4 is the main block.
//
type scopes []scope

// scope is a scope.
type scope struct {
	// fn is the node of the function that includes the scope.
	// nil in scopes 0, 1 and 2. It is also nil in scope 3 for scripts.
	fn *ast.Func
	// names are the declared names in the scope.
	names map[string]scopeEntry
}

// scopeEntry is a scope entry.
type scopeEntry struct {
	ti    *typeInfo       // type info.
	ident *ast.Identifier // declaration identifier. nil for predeclared identifiers (scopes 0, 1 and 2).
	impor *ast.Import     // import declaration. nil for not imported names.
	used  bool            // it has been used.
	param bool            // it is an in or out parameter of a function.
}

// newScopes returns a new scopes given the format types and the global block.
func newScopes(formats map[ast.Format]reflect.Type, global map[string]scopeEntry) scopes {
	var formatScope scope
	if len(formats) > 0 {
		formatScope = scope{names: map[string]scopeEntry{}}
		for f, t := range formats {
			name := formatTypeName[f]
			formatScope.names[name] = scopeEntry{ti: &typeInfo{
				Type:       t,
				Properties: propertyIsType | propertyIsFormatType | propertyUniverse,
			}}
		}
	}
	filePackageScope := scope{names: map[string]scopeEntry{}}
	return scopes{{names: universe}, formatScope, {names: global}, filePackageScope}
}

// Universe returns the type info of name as declared in the universe block
// and true. Otherwise it returns nil and false.
func (s scopes) Universe(name string) (*typeInfo, bool) {
	e, ok := universe[name]
	if ok {
		return e.ti, true
	}
	e, ok = s[1].names[name]
	return e.ti, ok
}

// Global returns the type info of name as declared in the global block and
// true. Otherwise it returns nil and false.
func (s scopes) Global(name string) (*typeInfo, bool) {
	e, ok := s[2].names[name]
	return e.ti, ok
}

// FilePackage returns the type info of name as declared in the file/package
// block and true. Otherwise it returns nil and false.
func (s scopes) FilePackage(name string) (*typeInfo, bool) {
	e, ok := s[3].names[name]
	return e.ti, ok
}

// Current returns the identifier of name as declared in the current scope and
// true. Otherwise it returns nil and false.
func (s scopes) Current(name string) (*ast.Identifier, bool) {
	e, ok := s[len(s)-1].names[name]
	return e.ident, ok
}

// FilePackageNames returns the names declared in the file/package block.
func (s scopes) FilePackageNames() []string {
	names := make([]string, len(s[3].names))
	i := 0
	for n := range s[3].names {
		names[i] = n
		i++
	}
	return names
}

// Declare declares name in the current scope with its type info and
// identifier and returns true. If name is already declared in the current
// scope, it does nothing and returns false.
func (s scopes) Declare(name string, ti *typeInfo, node ast.Node) bool {
	i := len(s) - 1
	e := scopeEntry{ti: ti}
	switch n := node.(type) {
	case *ast.Identifier:
		e.ident = n
	case *ast.Import:
		e.impor = n
	default:
		panic("scopes: invalid node type")
	}
	if names := s[i].names; names == nil {
		s[i].names = map[string]scopeEntry{name: e}
	} else if _, ok := names[name]; ok {
		return false
	} else {
		names[name] = e
	}
	return true
}

// Lookup lookups name in all scopes, and returns its type info, its
// identifier and true. Otherwise it returns nil, nil and false.
func (s scopes) Lookup(name string) (*typeInfo, *ast.Identifier, bool) {
	e, i := s.lookup(name, 0)
	return e.ti, e.ident, i != -1
}

// LookupInFunc lookups name in function scopes, including the main block in
// scripts, and returns its type info, its identifier and true. Otherwise it
// returns nil, nil and false.
func (s scopes) LookupInFunc(name string) (*typeInfo, *ast.Identifier, bool) {
	e, i := s.lookup(name, 4)
	return e.ti, e.ident, i != -1
}

// lookup lookups name and returns its entry and scope index in which it is
// defined. Otherwise it returns the zero value of scopeEntry and -1.
// start is the index of the scope from which to start the lookup.
func (s scopes) lookup(name string, start int) (scopeEntry, int) {
	for i := len(s) - 1; i >= start; i-- {
		if e, ok := s[i].names[name]; ok {
			return e, i
		}
	}
	return scopeEntry{}, -1
}

// SetAsUsed sets name as used.
func (s scopes) SetAsUsed(name string) {
	e, i := s.lookup(name, 3)
	if i != -1 && !e.used {
		e.used = true
		s[i].names[name] = e
	}
}

// UnusedVariable returns the declaration of the first unused variable, by
// position in the source, declared in the current scope. If all variables in
// the current scope are used, it returns nil.
func (s scopes) UnusedVariable() *ast.Identifier {
	var ident *ast.Identifier
	for _, e := range s[len(s)-1].names {
		if e.used || e.param || e.ti.IsConstant() || e.ti.IsType() {
			continue
		}
		if ident == nil || e.ident.Position.Start < ident.Start {
			ident = e.ident
		}
	}
	return ident
}

// UnusedImport returns the declaration of the first unused import, by
// position in the source. If all imports are used, it returns nil.
func (s scopes) UnusedImport() *ast.Import {
	unused := map[*ast.Import]bool{}
	for _, e := range s[len(s)-1].names {
		if e.impor == nil {
			continue
		}
		if e.ti.IsPackage() {
			if !e.used {
				unused[e.impor] = true
			}
			continue
		}
		if _, ok := unused[e.impor]; !ok || e.used {
			unused[e.impor] = !e.used
		}
	}
	var node *ast.Import
	for im, ok := range unused {
		if ok && (node == nil || im.Position.Start < node.Start) {
			node = im
		}
	}
	return node
}

// CurrentFunction returns the function of the current scope or nil if there
// is no function. There is no function for the main block of scripts.
func (s scopes) CurrentFunction() *ast.Func {
	return s[len(s)-1].fn
}

// Function returns the function in which name is declared. If name is not
// declared in a function, it returns nil.
func (s scopes) Function(name string) *ast.Func {
	if _, i := s.lookup(name, 4); i != -1 {
		return s[i].fn
	}
	return nil
}

// Functions returns all the functions up to the function of the current
// scope. If there is no function, it returns nil. There is no function for
// the main block of scripts.
func (s scopes) Functions() []*ast.Func {
	n := 0
	for i, sc := range s {
		if sc.fn != nil && sc.fn != s[i-1].fn {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	functions := make([]*ast.Func, 0, n)
	for i, sc := range s {
		if sc.fn != nil && sc.fn != s[i-1].fn {
			functions = append(functions, sc.fn)
		}
	}
	return functions
}

// Enter enters a new scope. For function scopes fn is the function node,
// otherwise it is nil.
func (s scopes) Enter(fn *ast.Func) scopes {
	if fn == nil {
		return append(s, scope{fn: s[len(s)-1].fn})
	}
	names := map[string]scopeEntry{}
	t := fn.Type.Reflect
	for i := 0; i < t.NumIn(); i++ {
		p := fn.Type.Parameters[i]
		if p.Ident != nil && p.Ident.Name != "_" {
			ti := &typeInfo{Type: t.In(i), Properties: propertyAddressable}
			names[p.Ident.Name] = scopeEntry{ti: ti, ident: p.Ident, param: true}
		}
	}
	for i := 0; i < t.NumOut(); i++ {
		p := fn.Type.Result[i]
		if p.Ident != nil && p.Ident.Name != "_" {
			ti := &typeInfo{Type: t.Out(i), Properties: propertyAddressable}
			names[p.Ident.Name] = scopeEntry{ti: ti, ident: p.Ident, param: true}
		}
	}
	return append(s, scope{fn: fn, names: names})
}

// Exit exits the current scope.
func (s scopes) Exit() scopes {
	last := len(s) - 1
	if last == 3 {
		panic("scopes: no scope to exit of")
	}
	return s[:last]
}

var boolType = reflect.TypeOf(false)
var uintType = reflect.TypeOf(uint(0))
var uint8Type = reflect.TypeOf(uint8(0))
var int32Type = reflect.TypeOf(int32(0))
var errorType = reflect.TypeOf((*error)(nil)).Elem()

var uint8TypeInfo = &typeInfo{Type: uint8Type, Properties: propertyIsType | propertyUniverse}
var int32TypeInfo = &typeInfo{Type: int32Type, Properties: propertyIsType | propertyUniverse}

// universe is the universe scope.
var universe = map[string]scopeEntry{
	"append":     {ti: &typeInfo{Properties: propertyUniverse}},
	"cap":        {ti: &typeInfo{Properties: propertyUniverse}},
	"close":      {ti: &typeInfo{Properties: propertyUniverse}},
	"complex":    {ti: &typeInfo{Properties: propertyUniverse}},
	"copy":       {ti: &typeInfo{Properties: propertyUniverse}},
	"delete":     {ti: &typeInfo{Properties: propertyUniverse}},
	"imag":       {ti: &typeInfo{Properties: propertyUniverse}},
	"iota":       {ti: &typeInfo{Properties: propertyUniverse, Type: intType}},
	"len":        {ti: &typeInfo{Properties: propertyUniverse}},
	"make":       {ti: &typeInfo{Properties: propertyUniverse}},
	"new":        {ti: &typeInfo{Properties: propertyUniverse}},
	"nil":        {ti: &typeInfo{Properties: propertyUntyped | propertyUniverse}},
	"panic":      {ti: &typeInfo{Properties: propertyUniverse}},
	"print":      {ti: &typeInfo{Properties: propertyUniverse}},
	"println":    {ti: &typeInfo{Properties: propertyUniverse}},
	"real":       {ti: &typeInfo{Properties: propertyUniverse}},
	"recover":    {ti: &typeInfo{Properties: propertyUniverse}},
	"byte":       {ti: uint8TypeInfo},
	"bool":       {ti: &typeInfo{Type: boolType, Properties: propertyIsType | propertyUniverse}},
	"complex128": {ti: &typeInfo{Type: complex128Type, Properties: propertyIsType | propertyUniverse}},
	"complex64":  {ti: &typeInfo{Type: complex64Type, Properties: propertyIsType | propertyUniverse}},
	"error":      {ti: &typeInfo{Type: errorType, Properties: propertyIsType | propertyUniverse}},
	"float32":    {ti: &typeInfo{Type: reflect.TypeOf(float32(0)), Properties: propertyIsType | propertyUniverse}},
	"float64":    {ti: &typeInfo{Type: float64Type, Properties: propertyIsType | propertyUniverse}},
	"false":      {ti: &typeInfo{Type: boolType, Properties: propertyUniverse | propertyUntyped, Constant: boolConst(false)}},
	"int":        {ti: &typeInfo{Type: intType, Properties: propertyIsType | propertyUniverse}},
	"int16":      {ti: &typeInfo{Type: reflect.TypeOf(int16(0)), Properties: propertyIsType | propertyUniverse}},
	"int32":      {ti: int32TypeInfo},
	"int64":      {ti: &typeInfo{Type: reflect.TypeOf(int64(0)), Properties: propertyIsType | propertyUniverse}},
	"int8":       {ti: &typeInfo{Type: reflect.TypeOf(int8(0)), Properties: propertyIsType | propertyUniverse}},
	"rune":       {ti: int32TypeInfo},
	"string":     {ti: &typeInfo{Type: stringType, Properties: propertyIsType | propertyIsFormatType | propertyUniverse}},
	"true":       {ti: &typeInfo{Type: boolType, Properties: propertyUniverse | propertyUntyped, Constant: boolConst(true)}},
	"uint":       {ti: &typeInfo{Type: uintType, Properties: propertyIsType | propertyUniverse}},
	"uint16":     {ti: &typeInfo{Type: reflect.TypeOf(uint16(0)), Properties: propertyIsType | propertyUniverse}},
	"uint32":     {ti: &typeInfo{Type: reflect.TypeOf(uint32(0)), Properties: propertyIsType | propertyUniverse}},
	"uint64":     {ti: &typeInfo{Type: reflect.TypeOf(uint64(0)), Properties: propertyIsType | propertyUniverse}},
	"uint8":      {ti: uint8TypeInfo},
	"uintptr":    {ti: &typeInfo{Type: reflect.TypeOf(uintptr(0)), Properties: propertyIsType | propertyUniverse}},
}
