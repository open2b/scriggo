// Copyright (c) 2021 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"

	"github.com/open2b/scriggo/compiler/ast"
)

var boolType = reflect.TypeOf(false)
var uintType = reflect.TypeOf(uint(0))
var uint8Type = reflect.TypeOf(uint8(0))
var int32Type = reflect.TypeOf(int32(0))
var errorType = reflect.TypeOf((*error)(nil)).Elem()

var uint8TypeInfo = &typeInfo{Type: uint8Type, Properties: propertyIsType | propertyUniverse}
var int32TypeInfo = &typeInfo{Type: int32Type, Properties: propertyIsType | propertyUniverse}

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

// scopes represents the universe block, global block, file block, package
// block and function scopes.
type scopes []scope

// scope is a scope.
type scope struct {
	// Function node that includes the scope. nil in scopes 0, 1 and 2. It is also nil in scope 3 for scrips.
	fn *ast.Func
	// Names declared in the scope.
	names map[string]scopeEntry
}

// scopeEntry is a scope entry.
type scopeEntry struct {
	ti    *typeInfo       // type info.
	decl  *ast.Identifier // declaration node. nil for scopes 0, 1 and 2. It is also nil in scope 3 if it is imported.
	used  bool            // it has been used.
	param bool            // it is an in or out parameter of a function.
}

// newScopes returns a new scopes given the format types and the global block.
func newScopes(formats map[ast.Format]reflect.Type, global map[string]scopeEntry) scopes {
	//
	//   0, 1 are the universe block. 1 is for the format types.
	//   2    is the global block, nil for programs.
	//   3    is the file/package block.
	//   4+   are the scopes in the functions. For scripts, 4 if the main block.
	//
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
	return scopes{scope{names: universe}, formatScope, scope{names: global}, {}}
}

// Universe returns the type info of the given name in the universe block and
// true. If the name does not exist, returns nil and false.
func (s scopes) Universe(name string) (*typeInfo, bool) {
	elem, ok := s[0].names[name]
	if ok {
		return elem.ti, true
	}
	elem, ok = s[1].names[name]
	return elem.ti, ok
}

// Global returns the type info of the given name in the global block and
// true. If the name does not exist, returns nil and false.
func (s scopes) Global(name string) (*typeInfo, bool) {
	elem, ok := s[2].names[name]
	return elem.ti, ok
}

// FilePackage returns the type info of the given name in the file/package
// block and true. If the name does not exist, returns nil and false.
func (s scopes) FilePackage(name string) (*typeInfo, bool) {
	elem, ok := s[3].names[name]
	return elem.ti, ok
}

// Current returns the type info of the given name in the current scope and
// true. If the name does not exist, returns nil and false.
func (s scopes) Current(name string) (*ast.Identifier, bool) {
	elem, ok := s[len(s)-1].names[name]
	return elem.decl, ok
}

// FilePackageNames returns the names in the file/package block.
func (s scopes) FilePackageNames() []string {
	names := make([]string, len(s[3].names))
	i := 0
	for n := range s[3].names {
		names[i] = n
		i++
	}
	return names
}

// IsImported reports whether name is in the file/package block and it is
// imported from a package or a template file.
func (s scopes) IsImported(name string) bool {
	elem, ok := s[3].names[name]
	return ok && elem.decl == nil
}

// IsParameter reports whether name is a function parameter.
func (s scopes) IsParameter(name string) bool {
	e, _ := s.lookup(name, 4)
	return e.param
}

// SetCurrent sets name in the current scope with the type info ti and
// declaration decl. param indicates if it is a function parameter.
func (s scopes) SetCurrent(name string, ti *typeInfo, decl *ast.Identifier, param bool) {
	n := len(s) - 1
	if s[n].names == nil {
		s[n].names = map[string]scopeEntry{}
	}
	s[n].names[name] = scopeEntry{ti: ti, decl: decl, param: param}
}

// SetFilePackage sets name in the file/package block with type info ti.
func (s scopes) SetFilePackage(name string, ti *typeInfo) {
	if s[3].names == nil {
		s[3].names = map[string]scopeEntry{}
	}
	s[3].names[name] = scopeEntry{ti: ti}
}

// Lookup lookups name and, if exists, returns its type info, the node of the
// identifier and true. Otherwise returns nil, nil and false.
func (s scopes) Lookup(name string) (*typeInfo, *ast.Identifier, bool) {
	e, ok := s.lookup(name, 0)
	return e.ti, e.decl, ok
}

// LookupInFunc lookups name and, if exists, returns the type info, the node
// of the identifier and true s. Otherwise returns nil, nil and
// false.
func (s scopes) LookupInFunc(name string) (*typeInfo, *ast.Identifier, bool) {
	e, ok := s.lookup(name, 4)
	return e.ti, e.decl, ok
}

// lookup lookups name and, if exists, returns the its entry and true.
// Otherwise scopeEntry{} and false.
func (s scopes) lookup(name string, start int) (scopeEntry, bool) {
	for i := len(s) - 1; i >= start; i-- {
		if e, ok := s[i].names[name]; ok {
			return e, true
		}
	}
	return scopeEntry{}, false
}

// SetAsUsed sets name as used.
func (s scopes) SetAsUsed(name string) {
	for i := len(s) - 1; i >= 4; i-- {
		if elem, ok := s[i].names[name]; ok {
			if !elem.used {
				elem.used = true
				s[i].names[name] = elem
			}
			return
		}
	}
}

// Unused returns the first, per source position, unused name in the current
// scope.
func (s scopes) Unused() (*ast.Identifier, bool) {
	var decl *ast.Identifier
	for _, elem := range s[len(s)-1].names {
		if elem.used || elem.param || elem.ti.IsConstant() || elem.ti.IsType() {
			continue
		}
		if decl == nil || elem.decl.Position.Start < decl.Start {
			decl = elem.decl
		}
	}
	return decl, decl != nil
}

// CurrentFunction returns the current function or nil if there is no
// function.
func (s scopes) CurrentFunction() *ast.Func {
	return s[len(s)-1].fn
}

// Function returns the function in which name is declared. Returns nil if it
// is not declared or it is not declared in a function.
func (s scopes) Function(name string) *ast.Func {
	for i := len(s) - 1; i >= 0; i-- {
		if _, ok := s[i].names[name]; ok {
			return s[i].fn
		}
	}
	return nil
}

// Functions returns all the functions up to the current scope.
// If there is no function, returns nil.
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

// Enter enters a new scope.
func (s scopes) Enter(fn *ast.Func) scopes {
	if fn == nil {
		fn = s[len(s)-1].fn
	}
	return append(s, scope{fn: fn})
}

// Exit exits the current scope.
func (s scopes) Exit() scopes {
	last := len(s) - 1
	if last == 3 {
		panic("scopes: no scope to exit of")
	}
	return s[:last]
}
