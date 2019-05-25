// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"
	"unicode"

	"scrigo/internal/compiler/ast"
	"scrigo/vm"
)

// AddExplicitReturn adds an explicit return statement as last statement to node.
func AddExplicitReturn(node ast.Node) {
	switch node := node.(type) {
	case *ast.Func:
		var pos *ast.Position
		if len(node.Body.Nodes) == 0 {
			pos = node.Pos()
		} else {
			last := node.Body.Nodes[len(node.Body.Nodes)-1]
			if _, ok := last.(*ast.Return); !ok {
				pos = last.Pos()
			}
		}
		if pos != nil {
			ret := ast.NewReturn(pos, nil)
			node.Body.Nodes = append(node.Body.Nodes, ret)
		}
	case *ast.Tree:
		var pos *ast.Position
		if len(node.Nodes) == 0 {
			pos = node.Pos()
		} else {
			last := node.Nodes[len(node.Nodes)-1]
			if _, ok := last.(*ast.Return); !ok {
				pos = last.Pos()
			}
		}
		if pos != nil {
			ret := ast.NewReturn(pos, nil)
			node.Nodes = append(node.Nodes, ret)
		}
	default:
		panic("bug") // TODO(Gianluca): remove.
	}
}

// changeRegister moves src content into dst, making a conversion if necessary.
func (e *Emitter) changeRegister(k bool, src, dst int8, srcType reflect.Type, dstType reflect.Type) {
	if kindToType(srcType.Kind()) != vm.TypeGeneral && dstType.Kind() == reflect.Interface {
		if k {
			e.FB.EnterStack()
			tmpReg := e.FB.NewRegister(srcType.Kind())
			e.FB.Move(true, src, tmpReg, srcType.Kind())
			e.FB.Convert(tmpReg, srcType, dst, srcType.Kind())
			e.FB.ExitStack()
		} else {
			e.FB.Convert(src, srcType, dst, srcType.Kind())
		}
	} else if k || src != dst {
		e.FB.Move(k, src, dst, srcType.Kind())
	}
}

// compositeLiteralLen returns node's length.
func compositeLiteralLen(node *ast.CompositeLiteral) int {
	size := 0
	for _, kv := range node.KeyValues {
		if kv.Key != nil {
			key := kv.Key.(*ast.Value).Val.(int)
			if key > size {
				size = key
			}
		}
		size++
	}
	return size
}

func (e *Emitter) importPredefinedPackage(n *ast.Import) {
	var importPkgName string
	parserPredefinedPkg := e.importablePredefinedPkgs[n.Path]
	if n.Ident == nil {
		importPkgName = parserPredefinedPkg.Name
	} else {
		switch n.Ident.Name {
		case "_":
			panic("TODO(Gianluca): not implemented")
		case ".":
			importPkgName = ""
		default:
			importPkgName = n.Ident.Name
		}
	}
	for ident, value := range parserPredefinedPkg.Declarations {
		_ = ident
		if _, ok := value.(reflect.Type); ok {
			continue
		}
		if reflect.TypeOf(value).Kind() == reflect.Ptr {
			e.globals = append(e.globals, vm.Global{Pkg: parserPredefinedPkg.Name, Name: ident, Value: value})
			name := ident
			if importPkgName != "" {
				name = importPkgName + "." + ident
			}
			e.globalNameIndex[e.currentPackage][name] = int16(len(e.globals) - 1)
		}
		if reflect.TypeOf(value).Kind() == reflect.Func {
			predefinedFunc := NewPredefinedFunction(parserPredefinedPkg.Name, ident, value)
			if importPkgName == "" {
				e.availablePredefinedFunctions[e.currentPackage][ident] = predefinedFunc
			} else {
				e.availablePredefinedFunctions[e.currentPackage][importPkgName+"."+ident] = predefinedFunc
			}
		}
	}
	e.isPredefinedPkg[importPkgName] = true
}

// isExported indicates if name is exported, according to
// https://golang.org/ref/spec#Exported_identifiers.
func isExported(name string) bool {
	return unicode.Is(unicode.Lu, []rune(name)[0])
}

// isLenBuiltinCall indicates if expr is a "len" builtin call.
func (e *Emitter) isLenBuiltinCall(expr ast.Expression) bool {
	if call, ok := expr.(*ast.Call); ok {
		if ti := e.TypeInfo[call]; ti.IsBuiltin() {
			if name := call.Func.(*ast.Identifier).Name; name == "len" {
				return true
			}
		}
	}
	return false
}

// isNil indicates if expr is the nil identifier.
func isNil(expr ast.Expression) bool {
	ident, ok := expr.(*ast.Identifier)
	if !ok {
		return false
	}
	return ident.Name == "nil"
}

// kindToType returns VM's type of k.
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

// mayHaveDepencencies indicates if there may be dependencies between values and
// variables.
func mayHaveDepencencies(variables, values []ast.Expression) bool {
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

// predefinedFunctionIndex returns fun's index inside current function,
// creating it if not exists.
func (e *Emitter) predefinedFunctionIndex(fun *vm.PredefinedFunction) int8 {
	currFun := e.CurrentFunction
	i, ok := e.assignedPredefinedFunctions[currFun][fun]
	if ok {
		return i
	}
	i = int8(len(currFun.Predefined))
	currFun.Predefined = append(currFun.Predefined, fun)
	if e.assignedPredefinedFunctions[currFun] == nil {
		e.assignedPredefinedFunctions[currFun] = make(map[*vm.PredefinedFunction]int8)
	}
	e.assignedPredefinedFunctions[currFun][fun] = i
	return i
}

// functionIndex returns fun's index inside current function, creating it if
// not exists.
func (e *Emitter) functionIndex(fun *vm.Function) int8 {
	currFun := e.CurrentFunction
	i, ok := e.assignedFunctions[currFun][fun]
	if ok {
		return i
	}
	i = int8(len(currFun.Functions))
	currFun.Functions = append(currFun.Functions, fun)
	if e.assignedFunctions[currFun] == nil {
		e.assignedFunctions[currFun] = make(map[*vm.Function]int8)
	}
	e.assignedFunctions[currFun][fun] = i
	return i
}

// setClosureRefs sets closure refs for function. This function works on current
// function builder, so shall be called before changing/saving it.
func (e *Emitter) setClosureRefs(fn *vm.Function, upvars []ast.Upvar) {

	// First: updates indexes of declarations that are found at the same level
	// of fn with appropriate register indexes.
	for i := range upvars {
		uv := &upvars[i]
		if uv.Index == -1 {
			name := uv.Declaration.(*ast.Identifier).Name
			reg := e.FB.ScopeLookup(name)
			uv.Index = int16(reg)
		}
	}

	// Second: updates upvarNames with external-defined names.
	closureRefs := make([]int16, len(upvars))
	e.upvarsNames[fn] = make(map[string]int)
	for i, uv := range upvars {
		e.upvarsNames[fn][uv.Declaration.(*ast.Identifier).Name] = i
		closureRefs[i] = uv.Index
	}

	// Third: var refs of current function are updated.
	fn.VarRefs = closureRefs

}
