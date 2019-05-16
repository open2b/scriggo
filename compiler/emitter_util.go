// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"reflect"

	"scrigo/compiler/ast"
	"scrigo/vm"
)

// addExplicitReturn adds an explicit return statement as last statement to node.
func addExplicitReturn(node ast.Node) {
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

func (c *Emitter) emitImport(n *ast.Import) {
	if n.Tree == nil { // Go package.
		var importPkgName string
		parserGoPkg := c.importableGoPkgs[n.Path]
		if n.Ident == nil {
			importPkgName = parserGoPkg.Name
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
		for ident, value := range parserGoPkg.Declarations {
			_ = ident
			if _, ok := value.(reflect.Type); ok {
				continue
			}
			if reflect.TypeOf(value).Kind() == reflect.Ptr {
				// pkg.DefineVariable(ident, value)
				// continue
				v := vm.Global{Pkg: parserGoPkg.Name, Name: ident, Value: value}
				if importPkgName == "" {
					c.availableVariables[ident] = v
				} else {
					c.availableVariables[importPkgName+"."+ident] = v
				}
			}
			if reflect.TypeOf(value).Kind() == reflect.Func {
				nativeFunc := NewNativeFunction(parserGoPkg.Name, ident, value)
				// index, ok := pkg.AddNativeFunction(nativeFunc)
				// if !ok {
				// 	panic("TODO: not implemented")
				// }
				// pkg.nativeFunctionsNames[ident] = int8(index)
				// continue
				if importPkgName == "" {
					c.availableNativeFunctions[ident] = nativeFunc
				} else {
					c.availableNativeFunctions[importPkgName+"."+ident] = nativeFunc
				}
			}
		}
		c.isNativePkg[importPkgName] = true
	} else {
		c.emitPackage(n.Tree.Nodes[0].(*ast.Package))
	}
}

// isLenBuiltinCall indicates if expr is a "len" builtin call.
func (c *Emitter) isLenBuiltinCall(expr ast.Expression) bool {
	if call, ok := expr.(*ast.Call); ok {
		if ti := c.typeinfo[call]; ti.IsBuiltin() {
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

// setClosureRefs sets closure refs for function. This function works on current
// function builder, so shall be called before changing/saving it.
func (c *Emitter) setClosureRefs(fn *vm.ScrigoFunction, upvars []ast.Upvar) {

	// First: updates indexes of declarations that are found at the same level
	// of fn with appropriate register indexes.
	for i := range upvars {
		uv := &upvars[i]
		if uv.Index == -1 {
			name := uv.Declaration.(*ast.Identifier).Name
			reg := c.fb.ScopeLookup(name)
			uv.Index = int16(reg)
		}
	}

	// Second: updates upvarNames with external-defined names.
	closureRefs := make([]int16, len(upvars))
	c.upvarsNames[fn] = make(map[string]int)
	for i, uv := range upvars {
		c.upvarsNames[fn][uv.Declaration.(*ast.Identifier).Name] = i
		closureRefs[i] = uv.Index
	}

	// Third: var refs of current function are updated.
	fn.VarRefs = closureRefs

}
