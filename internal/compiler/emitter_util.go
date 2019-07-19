// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
	"unicode"
	"unicode/utf8"

	"scriggo/ast"
	"scriggo/vm"
)

// changeRegister emits the code that move the content of register src to
// register dst, making a conversion if necessary.
func (em *emitter) changeRegister(k bool, src, dst int8, srcType reflect.Type, dstType reflect.Type) {

	// dst is indirect, so value must be "typed" to its true (original) type
	// before putting it into general.
	if dst < 0 {
		em.fb.emitTypify(k, srcType, src, dst)
		return
	}

	// When moving a value from general to general, value's type must be
	// updated.
	if dstType.Kind() == reflect.Interface && srcType.Kind() == reflect.Interface {
		em.fb.emitMove(k, src, dst, srcType.Kind())
		return
	}

	// When moving a value from int, float or string to general, value's type
	// must be "typed" to its true (original) type.
	if dstType.Kind() == reflect.Interface {
		em.fb.emitTypify(k, srcType, src, dst)
		return
	}

	// Source register is different than destination register: a conversion is
	// needed.
	if dstType.Kind() != srcType.Kind() {
		if k {
			em.fb.enterScope()
			tmp := em.fb.newRegister(srcType.Kind())
			em.fb.emitMove(true, src, tmp, srcType.Kind())
			em.fb.emitConvert(tmp, dstType, dst, srcType.Kind())
			em.fb.exitScope()
		}
		em.fb.emitConvert(src, dstType, dst, srcType.Kind())
		return
	}

	if k || src != dst {
		em.fb.emitMove(k, src, dst, srcType.Kind())
	}

}

// compositeLiteralLen returns the length of a composite literal.
func (em *emitter) compositeLiteralLen(node *ast.CompositeLiteral) int {
	size := 0
	for _, kv := range node.KeyValues {
		if kv.Key != nil {
			key := int(em.ti(kv.Key).Constant.int64())
			if key > size {
				size = key
			}
		}
		size++
	}
	return size
}

// functionIndex returns the index of a function inside the current function,
// creating it if it does not exist.
func (em *emitter) functionIndex(fun *vm.Function) int8 {
	i, ok := em.funcIndexes[em.fb.fn][fun]
	if ok {
		return i
	}
	i = int8(len(em.fb.fn.Functions))
	em.fb.fn.Functions = append(em.fb.fn.Functions, fun)
	if em.funcIndexes[em.fb.fn] == nil {
		em.funcIndexes[em.fb.fn] = make(map[*vm.Function]int8)
	}
	em.funcIndexes[em.fb.fn][fun] = i
	return i
}

// isExported reports whether name is exported, according to
// https://golang.org/ref/spec#Exported_identifiers.
func isExported(name string) bool {
	r, _ := utf8.DecodeRuneInString(name)
	return unicode.Is(unicode.Lu, r)
}

// isLenBuiltinCall reports whether expr is a call to the builtin "len".
func (em *emitter) isLenBuiltinCall(expr ast.Expression) bool {
	if call, ok := expr.(*ast.Call); ok {
		if ti := em.ti(call); ti.IsBuiltin() {
			if name := call.Func.(*ast.Identifier).Name; name == "len" {
				return true
			}
		}
	}
	return false
}

// kindToType returns the internal register type of a reflect kind.
func kindToType(k reflect.Kind) vm.Type {
	switch k {
	case reflect.Bool:
		return vm.TypeInt
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return vm.TypeInt
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return vm.TypeInt
	case reflect.Float32, reflect.Float64:
		return vm.TypeFloat
	case reflect.String:
		return vm.TypeString
	default:
		return vm.TypeGeneral
	}
}

// mayHaveDependencies reports whether there may be dependencies between
// values and variables.
func mayHaveDependencies(variables, values []ast.Expression) bool {
	// TODO(Gianluca): this function can be optimized, although for now
	// readability has been preferred.
	allDifferentIdentifiers := func() bool {
		names := make(map[string]bool)
		for _, v := range variables {
			ident, ok := v.(*ast.Identifier)
			if !ok {
				return false
			}
			_, alreadyPresent := names[ident.Name]
			if alreadyPresent {
				return false
			}
			names[ident.Name] = true
		}
		for _, v := range values {
			ident, ok := v.(*ast.Identifier)
			if !ok {
				return false
			}
			_, alreadyPresent := names[ident.Name]
			if alreadyPresent {
				return false
			}
			names[ident.Name] = true
		}
		return true
	}
	return !allDifferentIdentifiers()
}

// predVarIndex returns the index of a global variable in globals, adding it
// if it does not exist.
func (em *emitter) predVarIndex(v reflect.Value, predPkgName, name string) int16 {
	index, ok := em.predVarIndexes[em.fb.fn][v]
	if ok {
		return index
	}
	index = int16(len(em.globals))
	g := Global{Pkg: predPkgName, Name: name, Value: v.Interface(), Type: v.Type().Elem()}
	if em.predVarIndexes[em.fb.fn] == nil {
		em.predVarIndexes[em.fb.fn] = make(map[reflect.Value]int16)
	}
	em.globals = append(em.globals, g)
	em.predVarIndexes[em.fb.fn][v] = index
	return index
}

// predFuncIndex returns the index of a predefined function in the current
// function, adding it if it does not exist.
func (em *emitter) predFuncIndex(fn reflect.Value, predPkgName, name string) int8 {
	index, ok := em.predFunIndexes[em.fb.fn][fn]
	if ok {
		return index
	}
	index = int8(len(em.fb.fn.Predefined))
	f := newPredefinedFunction(predPkgName, name, fn.Interface())
	if em.predFunIndexes[em.fb.fn] == nil {
		em.predFunIndexes[em.fb.fn] = map[reflect.Value]int8{}
	}
	em.fb.fn.Predefined = append(em.fb.fn.Predefined, f)
	em.predFunIndexes[em.fb.fn][fn] = index
	return index
}

// sameRegType reports whether k1 and k2 have the same type of register in
// current implementation of the VM.
func sameRegType(k1, k2 reflect.Kind) bool {
	return kindToType(k1) == kindToType(k2)
}

// setClosureRefs sets the closure refs of a function. setClosureRefs operates
// on current function builder, so shall be called before changing or saving
// it.
func (em *emitter) setClosureRefs(fn *vm.Function, closureVars []ast.Upvar) {

	// First: update the indexes of the declarations that are found at the
	// same level of fn with appropriate register indexes.
	for i := range closureVars {
		v := &closureVars[i]
		if v.Index == -1 {
			name := v.Declaration.(*ast.Identifier).Name
			reg := em.fb.scopeLookup(name)
			v.Index = int16(reg)
		}
	}

	// Second: update closureVarRefs with external-defined names.
	closureRefs := make([]int16, len(closureVars))
	em.closureVarRefs[fn] = make(map[string]int)
	if em.isTemplate {
		// If it's a template, adds reserved global variables.
		closureRefs = append(closureRefs, 0, 1, 2)
	}
	for i, v := range closureVars {
		em.closureVarRefs[fn][v.Declaration.(*ast.Identifier).Name] = i
		closureRefs[i] = v.Index
	}

	// Third: var refs of current function are updated.
	fn.VarRefs = closureRefs

}
