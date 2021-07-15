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

var universe = scope{
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

type scopeElement struct {
	ti   *typeInfo
	decl *ast.Identifier
}

// scope represents a scope.
type scope map[string]scopeElement

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
func newScopes(universe, global scope) scopes {
	return scopes{universe, global, {}}
}

// universe returns the type info of the given name in the universe block and
// true. If the name does not exist, returns nil and false.
func (s scopes) universe(name string) (*typeInfo, bool) {
	elem, ok := s[0][name]
	return elem.ti, ok
}

// global returns the type info of the given name in the global block and
// true. If the name does not exist, returns nil and false.
func (s scopes) global(name string) (*typeInfo, bool) {
	elem, ok := s[1][name]
	return elem.ti, ok
}

// globals returns the global scope.
func (s scopes) globals() scope {
	return s[1]
}

// filePackage returns the type info of the given name in the file/package
// block and true. If the name does not exist, returns nil and false.
func (s scopes) filePackage(name string) (*typeInfo, bool) {
	elem, ok := s[2][name]
	return elem.ti, ok
}

// filePackageNames returns the names in the file/package block.
func (s scopes) filePackageNames() []string {
	names := make([]string, len(s[2]))
	i := 0
	for n := range s[2] {
		names[i] = n
		i++
	}
	return names
}

// isImport reports whether name is in the file/package block and is imported
// from a package or a template file.
func (s scopes) isImported(name string) bool {
	elem, ok := s[2][name]
	return ok && elem.decl == nil
}

// setFilePackage sets name in the file/package block with type info ti.
func (s scopes) setFilePackage(name string, ti *typeInfo) {
	s[2][name] = scopeElement{ti: ti}
}

// alreadyDeclared report whether name is already declared in the current
// scope and if it declared returns the identifier and true, otherwise returns
// nil and false.
func (s scopes) alreadyDeclared(name string) (*ast.Identifier, bool) {
	elem, ok := s[len(s)-1][name]
	return elem.decl, ok
}

// lookup lookups name in s and returns the type info and true if it exists.
// Otherwise returns nil and false.
func (s scopes) lookup(name string) (*typeInfo, bool) {
	for i := len(s) - 1; i >= 0; i-- {
		if elem, ok := s[i][name]; ok {
			return elem.ti, true
		}
	}
	return nil, false
}

// local returns the type info of the given name in the current local scope
// true. If the name does not exist, returns nil and false.
func (s scopes) setCurrent(name string) (*typeInfo, bool) {
	elem, ok := s[len(s)-1][name]
	return elem.ti, ok
}

// append appends a new local scope to s.
func (s scopes) append() scopes {
	return append(s, scope{})
}

// remove removes a last local scope from s.
func (s scopes) remove() scopes {
	last := len(s) - 1
	if last == 2 {
		panic("no local scope to drop")
	}
	return s[:last]
}
