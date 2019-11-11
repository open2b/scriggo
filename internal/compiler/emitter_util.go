// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"io"
	"reflect"
	"unicode"
	"unicode/utf8"

	"scriggo/ast"
	"scriggo/runtime"
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

// stackDifference returns the difference of registers between a and b.
func stackDifference(a, b runtime.StackShift) runtime.StackShift {
	return runtime.StackShift{
		a[0] - b[0],
		a[1] - b[1],
		a[2] - b[2],
		a[3] - b[3],
	}
}

// functionIndex returns the index of a function inside the current function,
// creating it if it does not exist.
func (em *emitter) functionIndex(fun *runtime.Function) int8 {
	i, ok := em.funcIndexes[em.fb.fn][fun]
	if ok {
		return i
	}
	i = int8(len(em.fb.fn.Functions))
	em.fb.fn.Functions = append(em.fb.fn.Functions, fun)
	if em.funcIndexes[em.fb.fn] == nil {
		em.funcIndexes[em.fb.fn] = make(map[*runtime.Function]int8)
	}
	em.funcIndexes[em.fb.fn][fun] = i
	return i
}

// getVarIndex returns the index of the variable v, if exists.
func (em *emitter) getVarIndex(v ast.Expression) (index int, ok bool) {

	if ti := em.ti(v); ti != nil && ti.IsPredefined() {
		// TODO: this is the new way of accessing predefined vars. Incrementally
		// integrate into Scriggo, then remove the other checks.
		if index, ok := em.predefinedVarRefs[em.fb.fn][ti.value.(*reflect.Value)]; ok {
			return index, true
		}
		// TODO: obsolete, remove:
		if sel, ok := v.(*ast.Selector); ok {
			index := em.predVarIndex(ti.value.(*reflect.Value), ti.PredefPackageName, sel.Ident)
			return int(index), true
		}
		// TODO: obsolete, remove:
		if ident, ok := v.(*ast.Identifier); ok {
			index := em.predVarIndex(ti.value.(*reflect.Value), ti.PredefPackageName, ident.Name)
			return int(index), true
		}
	}

	var name string
	switch v := v.(type) {
	case *ast.Identifier:
		name = v.Name
	case *ast.Selector:
		if ident, ok := v.Expr.(*ast.Identifier); ok {
			name = ident.Name + "." + v.Ident
		} else {
			return 0, false
		}
	default:
		return 0, false
	}

	// TODO: can these maps be unified in some way?
	if index, ok := em.closureVarRefs[em.fb.fn][name]; ok {
		return index, true
	}
	if index, ok := em.availableVarIndexes[em.pkg][name]; ok {
		return int(index), true
	}

	return 0, false

}

// isExported reports whether name is exported, according to
// https://golang.org/ref/spec#Exported_identifiers.
func isExported(name string) bool {
	r, _ := utf8.DecodeRuneInString(name)
	return unicode.Is(unicode.Lu, r)
}

// builtinCallName returns the name of the builtin function in a call
// expression. If expr is not a call expression or the function is not a
// builtin, it returns an empty string.
func (em *emitter) builtinCallName(expr ast.Expression) string {
	if call, ok := expr.(*ast.Call); ok && em.ti(call.Func).IsBuiltinFunction() {
		return call.Func.(*ast.Identifier).Name
	}
	return ""
}

// numOut reports the number of return parameters of call, if it is a function
// call. If is not, returns 0 and false.
func (em *emitter) numOut(call *ast.Call) (int, bool) {
	if ti := em.ti(call.Func); ti != nil && ti.Type != nil {
		if ti.Type.Kind() == reflect.Func {
			return ti.Type.NumOut(), true
		}
	}
	return 0, false
}

// kindToType returns the internal register type of a reflect kind.
func kindToType(k reflect.Kind) registerType {
	switch k {
	case reflect.Bool:
		return intRegister
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intRegister
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return intRegister
	case reflect.Float32, reflect.Float64:
		return floatRegister
	case reflect.String:
		return stringRegister
	default:
		return generalRegister
	}
}

// predVarIndex returns the index of a global variable in globals, adding it
// if it does not exist.
func (em *emitter) predVarIndex(v *reflect.Value, predPkgName, name string) int16 {
	if index, ok := em.predefinedVarRefs[em.fb.fn][v]; ok {
		return int16(index)
	}
	index := len(em.globals)
	g := Global{Pkg: predPkgName, Name: name, Type: v.Type().Elem()}
	if !v.IsNil() {
		g.Value = v.Interface()
	}
	if em.predefinedVarRefs[em.fb.fn] == nil {
		em.predefinedVarRefs[em.fb.fn] = make(map[*reflect.Value]int)
	}
	em.globals = append(em.globals, g)
	em.predefinedVarRefs[em.fb.fn][v] = index
	return int16(index)
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

// canEmitDirectly reports whether a value of kind k1 can be emitted directly
// into a register of kind k2 without the needing of passing from an
// intermediate register.
//
// The result of this function depends from the current implementation of the
// VM.
func canEmitDirectly(k1, k2 reflect.Kind) bool {
	// If the destination register has an interface kind, it's not possible to
	// emit the value directly: a typify may be necessary.
	if k2 == reflect.Interface {
		return false
	}
	// Functions and arrays are handled as special cases in VM.
	if k1 == reflect.Func ||
		k2 == reflect.Func ||
		k1 == reflect.Array ||
		k2 == reflect.Array ||
		k1 == reflect.Struct ||
		k2 == reflect.Struct {
		return false
	}
	return kindToType(k1) == kindToType(k2)
}

// setClosureRefs sets the closure refs of a function. setClosureRefs operates
// on current function builder, so shall be called before changing or saving
// it.
func (em *emitter) setClosureRefs(fn *runtime.Function, closureVars []ast.Upvar) {

	// First: update the indexes of the declarations that are found at the
	// same level of fn with appropriate register indexes.
	for i := range closureVars {
		v := &closureVars[i]
		if v.Index == -1 {
			if v.Declaration != nil {
				name := v.Declaration.(*ast.Identifier).Name
				reg := em.fb.scopeLookup(name)
				v.Index = int16(reg)
			} else {
				v.Index = em.predVarIndex(v.PredefinedValue, v.PredefinedPkg, v.PredefinedName)
			}
		}
	}

	// Second: update closureVarRefs with external-defined names.
	closureRefs := make([]int16, len(closureVars))
	em.closureVarRefs[fn] = make(map[string]int)
	if em.isTemplate {
		// If it's a template, adds reserved global variables.
		closureRefs = append(closureRefs, 0, 1, 2, 3)
	}
	for i, v := range closureVars {
		if v.Declaration != nil {
			em.closureVarRefs[fn][v.Declaration.(*ast.Identifier).Name] = i
		} else {
			if em.predefinedVarRefs[fn] == nil {
				em.predefinedVarRefs[fn] = map[*reflect.Value]int{}
			}
			em.predefinedVarRefs[fn][v.PredefinedValue] = i
		}
		closureRefs[i] = v.Index
	}

	// Third: var refs of current function are updated.
	fn.VarRefs = closureRefs

}

// renderFuncType is a reflect.Type that stores the type of the render function
// defined in the scriggo/template package.
//
// Keep in sync with scriggo/template.render.
//
var renderFuncType = reflect.FuncOf([]reflect.Type{
	reflect.PtrTo(reflect.TypeOf(runtime.Env{})), // _ *runtime.Env
	reflect.TypeOf((*io.Writer)(nil)).Elem(),     // out io.Writer
	reflect.TypeOf((*interface{})(nil)).Elem(),   // value interface{}
	reflect.TypeOf(ast.Context(0)),               // ctx ast.Context
}, nil, false)

// ioWriterWriteType is a reflect.Type that stores the type of the function
//
//		Write(p []byte) (n int, err error)
//
//  defined in the interface io.Writer.
//
var ioWriterWriteType = reflect.FuncOf(
	[]reflect.Type{
		reflect.TypeOf([]byte{}), // p []byte
	},
	[]reflect.Type{
		reflect.TypeOf(int(0)),               // n int
		reflect.TypeOf((*error)(nil)).Elem(), // err error
	},
	false,
)

// urlEscaperStartURLType is a reflect.Type that sotres the type of the method
//
//		func (w *urlEscaper) StartURL(quoted, isSet bool)
//
// defined on the type urlEscaper.
//
// Keep in sync with scriggo/template.urlEscaper.StartURL.
//
var urlEscaperStartURLType = reflect.FuncOf([]reflect.Type{
	reflect.TypeOf(bool(false)), // quoted bool
	reflect.TypeOf(bool(false)), // isSet bool
}, nil, false)
