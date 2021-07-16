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

type scopeEntry struct {
	ti    *typeInfo
	decl  *ast.Identifier
	used  bool
	param bool
}

type scope struct {
	fn    *ast.Func // nil in scopes 0, 1 and 2. For scripts it is also nil in scope 3.
	names map[string]scopeEntry
}

// scopes represents the universe block, global block, file block, package
// block and local scopes.
//
//   0  is the universe block
//   1  is the global block, nil for programs.
//   2  is the file and package block.
//   3+ are the local scopes.
//
type scopes []scope

// newScopes returns a new scopes given universe and global blocks.
func newScopes(universe, global map[string]scopeEntry) scopes {
	return scopes{scope{names: universe}, scope{names: global}, {}}
}

// Universe returns the type info of the given name in the universe block and
// true. If the name does not exist, returns nil and false.
func (s scopes) Universe(name string) (*typeInfo, bool) {
	elem, ok := s[0].names[name]
	return elem.ti, ok
}

// Global returns the type info of the given name in the global block and
// true. If the name does not exist, returns nil and false.
func (s scopes) Global(name string) (*typeInfo, bool) {
	elem, ok := s[1].names[name]
	return elem.ti, ok
}

// Globals returns the global scope.
func (s scopes) Globals() map[string]scopeEntry {
	return s[1].names
}

// FilePackage returns the type info of the given name in the file/package
// block and true. If the name does not exist, returns nil and false.
func (s scopes) FilePackage(name string) (*typeInfo, bool) {
	elem, ok := s[2].names[name]
	return elem.ti, ok
}

// FilePackageNames returns the names in the file/package block.
func (s scopes) FilePackageNames() []string {
	names := make([]string, len(s[2].names))
	i := 0
	for n := range s[2].names {
		names[i] = n
		i++
	}
	return names
}

// IsImported reports whether name is in the file/package block and is imported
// from a package or a template file.
func (s scopes) IsImported(name string) bool {
	elem, ok := s[2].names[name]
	return ok && elem.decl == nil
}

// IsParameter reports whether name is a function parameter.
func (s scopes) IsParameter(name string) bool {
	e, _ := s.lookup(name, 3)
	return e.param
}

// SetLocal sets name in the current local scope with the type info ti and
// declaration decl. param indicates if it is a function parameter.
func (s scopes) SetLocal(name string, ti *typeInfo, decl *ast.Identifier, param bool) {
	n := len(s) - 1
	if s[n].names == nil {
		s[n].names = map[string]scopeEntry{}
	}
	s[n].names[name] = scopeEntry{ti: ti, decl: decl, param: param}
}

// SetFilePackage sets name in the file/package block with type info ti.
func (s scopes) SetFilePackage(name string, ti *typeInfo) {
	if s[2].names == nil {
		s[2].names = map[string]scopeEntry{}
	}
	s[2].names[name] = scopeEntry{ti: ti}
}

// AlreadyDeclared report whether name is already declared in the current
// scope and if it declared returns the identifier and true, otherwise returns
// nil and false.
func (s scopes) AlreadyDeclared(name string) (*ast.Identifier, bool) {
	elem, ok := s[len(s)-1].names[name]
	return elem.decl, ok
}

// Lookup lookups name in s and returns the type info, the node of the
// identifier and true if it exists. Otherwise returns nil, nil and false.
func (s scopes) Lookup(name string) (*typeInfo, *ast.Identifier, bool) {
	e, ok := s.lookup(name, 0)
	return e.ti, e.decl, ok
}

// LookupInFunc lookups name in s and returns the type info, the node of the
// identifier and true if it exists. Otherwise returns nil, nil and false.
func (s scopes) LookupInFunc(name string) (*typeInfo, *ast.Identifier, bool) {
	e, ok := s.lookup(name, 3)
	return e.ti, e.decl, ok
}

// lookup lookups name in s and returns the type info, the node of the
// identifier and true if it exists. Otherwise returns nil, nil and false.
func (s scopes) lookup(name string, start int) (scopeEntry, bool) {
	for i := len(s) - 1; i >= start; i-- {
		if e, ok := s[i].names[name]; ok {
			return e, true
		}
	}
	return scopeEntry{}, false
}

// CurrentFunction returns the current function or nil if there is no
// function.
func (s scopes) CurrentFunction() *ast.Func {
	return s[len(s)-1].fn
}

// SetAsUsed sets name as used.
func (s scopes) SetAsUsed(name string) {
	for i := len(s) - 1; i >= 3; i-- {
		if elem, ok := s[i].names[name]; ok {
			if !elem.used {
				elem.used = true
				s[i].names[name] = elem
			}
			return
		}
	}
	//panic("used name not found")
}

// Unused returns the first Unused name in the current local scope.
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

// Function returns the function in which name is declared. Returns nil if it
// is not declared or is not declared in a function.
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

// Enter enters a new local scope.
func (s scopes) Enter(fn *ast.Func) scopes {
	if fn == nil {
		n := len(s) - 1
		//if n == 2 {
		//panic("missing function for the first local scope")
		//}
		fn = s[n].fn
	}
	return append(s, scope{fn: fn})
}

// Exit exits the last local scope.
func (s scopes) Exit() scopes {
	last := len(s) - 1
	if last == 2 {
		panic("no local scope to exit")
	}
	return s[:last]
}
