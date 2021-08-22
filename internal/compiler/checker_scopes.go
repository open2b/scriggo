// Copyright (c) 2021 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"

	"github.com/open2b/scriggo/ast"
)

// scopes represents the universe block, global block, file/package block and
// function scopes.
//
//   0, 1 are the universe block. 1 is for the format types.
//   2    is the global block, empty for programs.
//   3    is the file/package block.
//   4+   are the scopes in functions. For templates and scripts, 4 is the main block.
//
type scopes struct {
	s           []scope
	path        string
	allowUnused bool
}

// scope is a scope.
type scope struct {
	// fn is function, with the related labels, where the scope is located.
	fn scopeFunc
	// Position of the block of the scope.
	block *ast.Position
	// names are the declared names in the scope.
	names map[string]scopeName
}

// scopeFunc represents a function scope. Only scopes 4 onwards have a function scope.
type scopeFunc struct {
	// node is the node of the function.
	// It is nil in scopes 0, 1, 2 and 3. It is also nil in scope 4 for scripts.
	node *ast.Func
	// labels are the labels, declared or only used, in the scope of the function.
	// It is nil if there are no labels and it is always nil in scopes 0, 1, 2 and 3.
	labels map[string]scopeLabel
}

// scopeName is a scope name.
type scopeName struct {
	// ti is the type info.
	ti *typeInfo
	// ident is the identifier in the declaration node.
	// It is nil for predeclared identifiers (scopes 0, 1 and 2), packages and imported names (scope 3).
	ident *ast.Identifier
	// impor is the import declaration of a package or imported name (scope 3).
	impor *ast.Import
	// used indicates if it has been used.
	used bool
}

// scopeLabel is a label in a function scope.
type scopeLabel struct {
	// Position of the block in which the label is declared.
	block *ast.Position
	// node is the node of the label.
	node *ast.Label
	// used indicates if it has been used.
	used bool
	// goto statements referring the label.
	gotos []scopeGoto
}

// scopeGoto represents a goto statement referring a label that has not yet
// been declared.
type scopeGoto struct {
	// pos is the position of the label identifier in the goto statement.
	pos *ast.Position
	// blocks are the positions of the goto blocks (with scope >= 4).
	blocks []*ast.Position
}

// newScopes returns a new scopes given the format types and the global block.
func newScopes(path string, formats map[ast.Format]reflect.Type, global map[string]scopeName) *scopes {
	// TODO(marco): change type: fn(string, map[string]*typeInfo, global map[string]*typeInfo)
	var formatScope scope
	if len(formats) > 0 {
		formatScope = scope{names: map[string]scopeName{}}
		for f, t := range formats {
			name := formatTypeName[f]
			formatScope.names[name] = scopeName{ti: &typeInfo{
				Type:       t,
				Properties: propertyIsType | propertyIsFormatType | propertyUniverse,
			}}
		}
	}
	return &scopes{
		path: path,
		s:    []scope{{names: universe}, formatScope, {names: global}, {names: map[string]scopeName{}}},
	}
}

// AllowUnused does not check unused variables and labels.
func (scopes *scopes) AllowUnused() {
	scopes.allowUnused = true
}

// Universe returns the type info of name as declared in the universe block
// and true. Otherwise it returns nil and false.
func (scopes *scopes) Universe(name string) (*typeInfo, bool) {
	n, ok := universe[name]
	if ok {
		return n.ti, true
	}
	n, ok = scopes.s[1].names[name]
	return n.ti, ok
}

// Global returns the type info of name as declared in the global block and
// true. Otherwise it returns nil and false.
func (scopes *scopes) Global(name string) (*typeInfo, bool) {
	n, ok := scopes.s[2].names[name]
	return n.ti, ok
}

// FilePackage returns the type info of name as declared in the file/package
// block and true. Otherwise it returns nil and false.
func (scopes *scopes) FilePackage(name string) (*typeInfo, bool) {
	n, ok := scopes.s[3].names[name]
	return n.ti, ok
}

// Current returns the identifier of name as declared in the current scope and
// true. Otherwise it returns nil and false.
func (scopes *scopes) Current(name string) (*ast.Identifier, bool) {
	n, ok := scopes.s[len(scopes.s)-1].names[name]
	return n.ident, ok
}

// FilePackageNames returns the names declared in the file/package block.
func (scopes *scopes) FilePackageNames() []string {
	names := make([]string, len(scopes.s[3].names))
	i := 0
	for n := range scopes.s[3].names {
		names[i] = n
		i++
	}
	return names
}

// Declare declares name in with its type info and node and returns true. If
// name is already declared in the current scope, it does nothing and returns
// false.
//
// node is an *ast.Import value for packages and imported names, otherwise nil
// for predeclared names, otherwise an *ast.Identifier value for all other
// names.
func (scopes *scopes) Declare(name string, ti *typeInfo, node ast.Node) bool {
	c := len(scopes.s) - 1
	n := scopeName{ti: ti}
	switch node := node.(type) {
	case nil:
	case *ast.Identifier:
		n.ident = node
	case *ast.Import:
		n.impor = node
	default:
		panic("scopes: invalid node type")
	}
	if names := scopes.s[c].names; names == nil {
		scopes.s[c].names = map[string]scopeName{name: n}
	} else if _, ok := names[name]; ok {
		return false
	} else {
		names[name] = n
	}
	return true
}

// DeclareLabel declares a label.
func (scopes *scopes) DeclareLabel(label *ast.Label) {

	c := len(scopes.s) - 1
	current := scopes.s[c]
	name := label.Ident.Name

	lbl, used := current.fn.labels[name]
	lbl.block = current.block
	lbl.node = label
	current.fn.labels[name] = lbl
	if !used {
		return
	}

	for _, g := range lbl.gotos {
		if i := c - 4; len(g.blocks) <= i || g.blocks[i] != current.block {
			panic(checkError(scopes.path, g.pos, "goto %s jumps into block starting at %s:%s",
				name, scopes.path, current.block))
		}
		var v *ast.Identifier
		s := g.pos.Start
		for _, n := range current.names {
			if n.ti.Addressable() && n.ident.Position.Start > s {
				v = n.ident
				s = n.ident.Position.Start
			}
		}
		if v != nil {
			panic(checkError(scopes.path, g.pos, "goto %s jumps over declaration of %s at %s:%s",
				name, v.Name, scopes.path, v.Position))
		}
	}

	return
}

// Lookup lookups name in all scopes, and returns its type info, its
// identifier and true. Otherwise it returns nil, nil and false.
func (scopes *scopes) Lookup(name string) (*typeInfo, *ast.Identifier, bool) {
	n, i := scopes.lookup(name, 0)
	return n.ti, n.ident, i != -1
}

// LookupInFunc lookups name in function scopes, including the main block in
// scripts, and returns its type info, its identifier and true. Otherwise it
// returns nil, nil and false.
func (scopes *scopes) LookupInFunc(name string) (*typeInfo, *ast.Identifier, bool) {
	n, i := scopes.lookup(name, 4)
	return n.ti, n.ident, i != -1
}

// LookupLabel lookups a label in the current function scope and returns the
// label node and true if it is declared. Otherwise it returns nil and false.
func (scopes *scopes) LookupLabel(name string) (*ast.Label, bool) {
	c := len(scopes.s) - 1
	lbl := scopes.s[c].fn.labels[name]
	if lbl.node == nil {
		return nil, false
	}
	return lbl.node, true
}

// lookup lookups name and returns its scope name and scope index in which it
// is defined. Otherwise it returns the zero value of scopeName and -1.
// start is the index of the scope from which to start the lookup.
func (scopes *scopes) lookup(name string, start int) (scopeName, int) {
	for i := len(scopes.s) - 1; i >= start; i-- {
		if n, ok := scopes.s[i].names[name]; ok {
			return n, i
		}
	}
	return scopeName{}, -1
}

// Use sets name as used and returns a boolean indicating whether it was
// already set as used.
func (scopes *scopes) Use(name string) bool {
	n, i := scopes.lookup(name, 3)
	if i != -1 && !n.used {
		n.used = true
		scopes.s[i].names[name] = n
		return false
	}
	return true
}

// UseLabel sets the label ident as used. stmt is "goto", "break" or
// "continue". Returns the label node or nil if ident is nil.
func (scopes *scopes) UseLabel(stmt string, ident *ast.Identifier) *ast.Label {

	if ident == nil {
		return nil
	}

	c := len(scopes.s) - 1
	current := scopes.s[c]
	name := ident.Name

	lbl := current.fn.labels[name]
	defer func() {
		current.fn.labels[name] = lbl
	}()
	lbl.used = true

	if stmt == "goto" {
		if lbl.node == nil {
			blocks := make([]*ast.Position, len(scopes.s)-4)
			for i, n := range scopes.s[4:] {
				blocks[i] = n.block
			}
			lbl.gotos = append(lbl.gotos, scopeGoto{pos: ident.Position, blocks: blocks})
			return nil
		}
		for i := c; i >= 4; i-- {
			if scopes.s[i].block == lbl.block {
				return lbl.node
			}
		}
		panic(checkError(scopes.path, lbl.block, "goto %s jumps into block starting at :%s", name, lbl.block))
	}
	if lbl.node == nil {
		panic(checkError(scopes.path, ident, "%s label not defined: %s", stmt, name))
	}

	return lbl.node
}

// UnusedImport returns the declaration of the first unused import, by
// position in the source. If all imports are used, it returns nil.
func (scopes *scopes) UnusedImport() *ast.Import {
	unused := map[*ast.Import]bool{}
	for _, n := range scopes.s[3].names {
		if n.impor == nil {
			continue
		}
		if n.ti.IsPackage() {
			if !n.used {
				unused[n.impor] = true
			}
			continue
		}
		if _, ok := unused[n.impor]; !ok || n.used {
			unused[n.impor] = !n.used
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
func (scopes *scopes) CurrentFunction() *ast.Func {
	c := len(scopes.s) - 1
	return scopes.s[c].fn.node
}

// Function returns the function in which name is declared. If name is not
// declared in a function, it returns nil.
func (scopes *scopes) Function(name string) *ast.Func {
	if _, i := scopes.lookup(name, 4); i != -1 {
		return scopes.s[i].fn.node
	}
	return nil
}

// Functions returns all the functions up to the function of the current
// scope. If there is no function, it returns nil. There is no function for
// the main block of scripts.
func (scopes *scopes) Functions() []*ast.Func {
	n := 0
	for i, sc := range scopes.s {
		if sc.fn.node != nil && sc.fn.node != scopes.s[i-1].fn.node {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	functions := make([]*ast.Func, 0, n)
	for i, sc := range scopes.s {
		if sc.fn.node != nil && sc.fn.node != scopes.s[i-1].fn.node {
			functions = append(functions, sc.fn.node)
		}
	}
	return functions
}

// Enter enters a new scope. block is the block of the scope.
func (scopes *scopes) Enter(block ast.Node) {
	s := scope{block: block.Pos()}
	if node, ok := block.(*ast.Func); ok {
		s.fn.node = node
	} else {
		c := len(scopes.s) - 1
		s.fn = scopes.s[c].fn
	}
	if s.fn.labels == nil {
		s.fn.labels = map[string]scopeLabel{}
	}
	scopes.s = append(scopes.s, s)
}

// isFuncBlock reports whether the current scope if a function block scope.
func (scopes *scopes) isFuncBlock() bool {
	c := len(scopes.s) - 1
	return c == 4 || c > 4 && scopes.s[c].fn.node != scopes.s[c-1].fn.node
}

// Exit exits the current scope.
func (scopes *scopes) Exit() {

	c := len(scopes.s) - 1

	// Check for goto statements referring non-defined labels.
	if scopes.isFuncBlock() {
		for name, lbl := range scopes.s[c].fn.labels {
			if lbl.node == nil {
				panic(checkError(scopes.path, lbl.gotos[0].pos, "label %s not defined", name))
			}
		}
	}

	if !scopes.allowUnused {

		// Check for unused labels.
		if scopes.isFuncBlock() {
			var label *ast.Label
			for _, lbl := range scopes.s[c].fn.labels {
				if !lbl.used && (label == nil || lbl.node.Position.Start < label.Start) {
					label = lbl.node
				}
			}
			if label != nil {
				panic(checkError(scopes.path, label, "label %s defined and not used", label.Ident))
			}
		}

		// Check for unused variables.
		var ident *ast.Identifier
		for _, n := range scopes.s[c].names {
			if n.used || n.ti.IsConstant() || n.ti.IsType() {
				continue
			}
			if ident == nil || n.ident.Position.Start < ident.Start {
				ident = n.ident
			}
		}
		if ident != nil {
			panic(checkError(scopes.path, ident, "%s declared but not used", ident))
		}

	}

	// Exit the scope.
	last := len(scopes.s) - 1
	if last == 3 {
		panic("scopes: no scope to exit of")
	}
	scopes.s = scopes.s[:last]

	return
}

var boolType = reflect.TypeOf(false)
var uintType = reflect.TypeOf(uint(0))
var uint8Type = reflect.TypeOf(uint8(0))
var int32Type = reflect.TypeOf(int32(0))
var errorType = reflect.TypeOf((*error)(nil)).Elem()

// universe is the universe scope.
var universe = map[string]scopeName{
	"append":     {ti: &typeInfo{Properties: propertyUniverse}},
	"cap":        {ti: &typeInfo{Properties: propertyUniverse}},
	"close":      {ti: &typeInfo{Properties: propertyUniverse}},
	"complex":    {ti: &typeInfo{Properties: propertyUniverse}},
	"copy":       {ti: &typeInfo{Properties: propertyUniverse}},
	"delete":     {ti: &typeInfo{Properties: propertyUniverse}},
	"imag":       {ti: &typeInfo{Properties: propertyUniverse}},
	"iota":       {ti: &typeInfo{Properties: propertyUniverse, Type: intType}},
	"itea":       {ti: &typeInfo{Properties: propertyUniverse | propertyUntyped | propertyAddressable}},
	"len":        {ti: &typeInfo{Properties: propertyUniverse}},
	"make":       {ti: &typeInfo{Properties: propertyUniverse}},
	"new":        {ti: &typeInfo{Properties: propertyUniverse}},
	"nil":        {ti: &typeInfo{Properties: propertyUntyped | propertyUniverse}},
	"panic":      {ti: &typeInfo{Properties: propertyUniverse}},
	"print":      {ti: &typeInfo{Properties: propertyUniverse}},
	"println":    {ti: &typeInfo{Properties: propertyUniverse}},
	"real":       {ti: &typeInfo{Properties: propertyUniverse}},
	"recover":    {ti: &typeInfo{Properties: propertyUniverse}},
	"byte":       {ti: &typeInfo{Type: uint8Type, Alias: "byte", Properties: propertyIsType | propertyUniverse}},
	"bool":       {ti: &typeInfo{Type: boolType, Properties: propertyIsType | propertyUniverse}},
	"complex128": {ti: &typeInfo{Type: complex128Type, Properties: propertyIsType | propertyUniverse}},
	"complex64":  {ti: &typeInfo{Type: complex64Type, Properties: propertyIsType | propertyUniverse}},
	"error":      {ti: &typeInfo{Type: errorType, Properties: propertyIsType | propertyUniverse}},
	"float32":    {ti: &typeInfo{Type: reflect.TypeOf(float32(0)), Properties: propertyIsType | propertyUniverse}},
	"float64":    {ti: &typeInfo{Type: float64Type, Properties: propertyIsType | propertyUniverse}},
	"false":      {ti: &typeInfo{Type: boolType, Properties: propertyUniverse | propertyUntyped, Constant: boolConst(false)}},
	"int":        {ti: &typeInfo{Type: intType, Properties: propertyIsType | propertyUniverse}},
	"int16":      {ti: &typeInfo{Type: reflect.TypeOf(int16(0)), Properties: propertyIsType | propertyUniverse}},
	"int32":      {ti: &typeInfo{Type: int32Type, Properties: propertyIsType | propertyUniverse}},
	"int64":      {ti: &typeInfo{Type: reflect.TypeOf(int64(0)), Properties: propertyIsType | propertyUniverse}},
	"int8":       {ti: &typeInfo{Type: reflect.TypeOf(int8(0)), Properties: propertyIsType | propertyUniverse}},
	"rune":       {ti: &typeInfo{Type: int32Type, Alias: "rune", Properties: propertyIsType | propertyUniverse}},
	"string":     {ti: &typeInfo{Type: stringType, Properties: propertyIsType | propertyIsFormatType | propertyUniverse}},
	"true":       {ti: &typeInfo{Type: boolType, Properties: propertyUniverse | propertyUntyped, Constant: boolConst(true)}},
	"uint":       {ti: &typeInfo{Type: uintType, Properties: propertyIsType | propertyUniverse}},
	"uint16":     {ti: &typeInfo{Type: reflect.TypeOf(uint16(0)), Properties: propertyIsType | propertyUniverse}},
	"uint32":     {ti: &typeInfo{Type: reflect.TypeOf(uint32(0)), Properties: propertyIsType | propertyUniverse}},
	"uint64":     {ti: &typeInfo{Type: reflect.TypeOf(uint64(0)), Properties: propertyIsType | propertyUniverse}},
	"uint8":      {ti: &typeInfo{Type: uint8Type, Properties: propertyIsType | propertyUniverse}},
	"uintptr":    {ti: &typeInfo{Type: reflect.TypeOf(uintptr(0)), Properties: propertyIsType | propertyUniverse}},
}
