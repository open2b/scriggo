// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
	"unicode"

	"scriggo/internal/compiler/ast"
	"scriggo/vm"
)

// changeRegister moves src content into dst, making a conversion if necessary.
func (e *emitter) changeRegister(k bool, src, dst int8, srcType reflect.Type, dstType reflect.Type) {

	// dst is indirect, so value must be "typed" to its true (original) type
	// before putting it into general.
	if dst < 0 {
		e.fb.Typify(k, srcType, src, dst)
		return
	}

	// When moving a value from general to general, value's type must be
	// updated.
	if dstType.Kind() == reflect.Interface && srcType.Kind() == reflect.Interface {
		e.fb.Move(k, src, dst, srcType.Kind())
		return
	}

	// When moving a value from int, float or string to general, value's type
	// must be "typed" to its true (original) type.
	if dstType.Kind() == reflect.Interface {
		e.fb.Typify(k, srcType, src, dst)
		return
	}

	// Source register is different than destination register: a convertion is
	// needed.
	if dstType.Kind() != srcType.Kind() {
		if k {
			e.fb.EnterScope()
			tmpReg := e.fb.NewRegister(srcType.Kind())
			e.fb.Move(true, src, tmpReg, srcType.Kind())
			e.fb.Convert(tmpReg, dstType, dst, srcType.Kind())
			e.fb.ExitScope()
		}
		e.fb.Convert(src, dstType, dst, srcType.Kind())
		return
	}

	if k || src != dst {
		e.fb.Move(k, src, dst, srcType.Kind())
	}
}

// compositeLiteralLen returns the length of a composite literal.
func (e *emitter) compositeLiteralLen(node *ast.CompositeLiteral) int {
	size := 0
	for _, kv := range node.KeyValues {
		if kv.Key != nil {
			key := int(e.typeInfos[kv.Key].Constant.int64())
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
func (e *emitter) functionIndex(fun *vm.Function) int8 {
	i, ok := e.assignedFuncs[e.fb.fn][fun]
	if ok {
		return i
	}
	i = int8(len(e.fb.fn.Functions))
	e.fb.fn.Functions = append(e.fb.fn.Functions, fun)
	if e.assignedFuncs[e.fb.fn] == nil {
		e.assignedFuncs[e.fb.fn] = make(map[*vm.Function]int8)
	}
	e.assignedFuncs[e.fb.fn][fun] = i
	return i
}

// isExported reports whether name is exported, according to
// https://golang.org/ref/spec#Exported_identifiers.
func isExported(name string) bool {
	return unicode.Is(unicode.Lu, []rune(name)[0])
}

// isLenBuiltinCall reports whether expr is a call to the builtin "len".
func (e *emitter) isLenBuiltinCall(expr ast.Expression) bool {
	if call, ok := expr.(*ast.Call); ok {
		if ti := e.typeInfos[call]; ti.IsBuiltin() {
			if name := call.Func.(*ast.Identifier).Name; name == "len" {
				return true
			}
		}
	}
	return false
}

// isNil reports whether expr is the nil identifier.
func isNil(expr ast.Expression) bool {
	// TODO(Gianluca): this implementation is wrong: nil can be shadowed. Use
	// typeinfo informations instead.
	if ident, ok := expr.(*ast.Identifier); ok {
		return ident.Name == "nil"
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
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
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

// predefVarIndex returns the index of a global variable in globals, adding it
// if it does not exist.
func (e *emitter) predefVarIndex(varRv reflect.Value, predefPkgName, name string) int16 {
	index, ok := e.predefVarIndexes[e.fb.fn][varRv]
	if ok {
		return index
	}
	index = int16(len(e.globals))
	g := Global{Pkg: predefPkgName, Name: name, Value: varRv.Interface(), Type: varRv.Type().Elem()}
	if e.predefVarIndexes[e.fb.fn] == nil {
		e.predefVarIndexes[e.fb.fn] = make(map[reflect.Value]int16)
	}
	e.globals = append(e.globals, g)
	e.predefVarIndexes[e.fb.fn][varRv] = index
	return index
}

// predefFuncIndex returns the index of a predefined function in the current
// function, adding it if it does not exist.
func (e *emitter) predefFuncIndex(funRv reflect.Value, predefPkgName, name string) int8 {
	index, ok := e.predefFunIndexes[e.fb.fn][funRv]
	if ok {
		return index
	}
	index = int8(len(e.fb.fn.Predefined))
	f := newPredefinedFunction(predefPkgName, name, funRv.Interface())
	if e.predefFunIndexes[e.fb.fn] == nil {
		e.predefFunIndexes[e.fb.fn] = make(map[reflect.Value]int8)
	}
	e.fb.fn.Predefined = append(e.fb.fn.Predefined, f)
	e.predefFunIndexes[e.fb.fn][funRv] = index
	return index
}

// setClosureRefs sets the closure refs of a function. setClosureRefs operates
// on current function builder, so shall be called before changing or saving
// it.
func (e *emitter) setClosureRefs(fn *vm.Function, upvars []ast.Upvar) {

	// First: updates indexes of declarations that are found at the same level
	// of fn with appropriate register indexes.
	for i := range upvars {
		uv := &upvars[i]
		if uv.Index == -1 {
			name := uv.Declaration.(*ast.Identifier).Name
			reg := e.fb.ScopeLookup(name)
			uv.Index = int16(reg)
		}
	}

	// Second: updates upvarNames with external-defined names.
	closureRefs := make([]int16, len(upvars))
	e.upvarsNames[fn] = make(map[string]int)
	if e.isTemplate {
		// If it's a template, adds reserved global variables.
		closureRefs = append(closureRefs, 0, 1, 2)
	}
	for i, uv := range upvars {
		e.upvarsNames[fn][uv.Declaration.(*ast.Identifier).Name] = i
		closureRefs[i] = uv.Index
	}

	// Third: var refs of current function are updated.
	fn.VarRefs = closureRefs

}
