// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"scrigo/internal/compiler/ast"
	"scrigo/native"
)

func ToTypeCheckerScope(gp *native.GoPackage) TypeCheckerScope {
	s := make(TypeCheckerScope, len(gp.Declarations))
	for ident, value := range gp.Declarations {
		// Importing a Go type.
		if t, ok := value.(reflect.Type); ok {
			s[ident] = scopeElement{t: &TypeInfo{
				Type:       t,
				Properties: PropertyIsType | PropertyIsNative,
			}}
			continue
		}
		// Importing a Go variable.
		if reflect.TypeOf(value).Kind() == reflect.Ptr {
			s[ident] = scopeElement{t: &TypeInfo{
				Type:       reflect.TypeOf(value).Elem(),
				Properties: PropertyAddressable | PropertyIsNative,
			}}
			continue
		}
		// Importing a Go global function.
		if reflect.TypeOf(value).Kind() == reflect.Func {
			s[ident] = scopeElement{t: &TypeInfo{
				Type:       reflect.TypeOf(value),
				Properties: PropertyIsNative,
			}}
			continue
		}
		// Importing a Go constant.
		s[ident] = scopeElement{t: &TypeInfo{
			Value:      value, // TODO (Gianluca): to review.
			Properties: PropertyIsConstant | PropertyIsNative,
		}}
	}
	return s
}

type PackageInfo struct {
	Name                 string
	Declarations         map[string]*TypeInfo
	ConstantsExpressions map[ast.Node]interface{} // expressions of constants.
	IndirectVars         map[*ast.Identifier]bool
	TypeInfo             map[ast.Node]*TypeInfo
}

func (pi *PackageInfo) String() string {
	s := "{\n"
	s += "\tName:                 " + pi.Name + "\n"
	s += "\tDeclarations:\n"
	for i, d := range pi.Declarations {
		s += fmt.Sprintf("                              %s: %s\n", i, d)
	}
	// TODO (Gianluca): add constant expressions.
	s += "}\n"
	return s
}

func depsOf(name string, deps GlobalsDependencies) []*ast.Identifier {
	for g, d := range deps {
		if g.Name == name {
			return d
		}
	}
	return nil
}

func checkDepsPath(path []*ast.Identifier, deps GlobalsDependencies) []*ast.Identifier {
	last := path[len(path)-1]
	for _, dep := range depsOf(last.Name, deps) {
		for _, p := range path {
			if p.Name == dep.Name {
				return append(path, dep)
			}
		}
		loopPath := checkDepsPath(append(path, dep), deps)
		if loopPath != nil {
			return loopPath
		}
	}
	return nil
}

func detectConstantsLoop(consts []*ast.Const, deps GlobalsDependencies) error {
	for _, c := range consts {
		path := []*ast.Identifier{c.Identifiers[0]}
		loopPath := checkDepsPath(path, deps)
		if loopPath != nil {
			msg := "constant definition loop\n"
			for i := 0; i < len(loopPath)-1; i++ {
				msg += "\t" + loopPath[i].Pos().String() + ": "
				msg += loopPath[i].String() + " uses " + loopPath[i+1].String()
				msg += "\n"
			}
			return errors.New(msg)
		}
	}
	return nil
}

func detectVarsLoop(vars []*ast.Var, deps GlobalsDependencies) error {
	for _, v := range vars {
		for _, left := range v.Lhs {
			path := []*ast.Identifier{left}
			loopPath := checkDepsPath(path, deps)
			if loopPath != nil {
				msg := "typechecking loop involving " + v.String() + "\n"
				for _, p := range loopPath {
					msg += "\t" + p.Pos().String() + ": " + p.String() + "\n"
				}
				return errors.New(msg)
			}
		}
	}
	return nil
}

func sortDeclarations(pkg *ast.Package, deps GlobalsDependencies) error {
	consts := []*ast.Const{}
	vars := []*ast.Var{}
	imports := []*ast.Import{}
	funcs := []*ast.Func{}

	// Fragments global declarations.
	for _, decl := range pkg.Declarations {
		switch decl := decl.(type) {
		case *ast.Import:
			imports = append(imports, decl)
		case *ast.Func:
			funcs = append(funcs, decl)
		case *ast.Const:
			if len(decl.Values) != len(decl.Identifiers) {
				panic("TODO(Gianluca): not implemented")
			}
			for i := range decl.Identifiers {
				consts = append(consts, ast.NewConst(decl.Pos(), decl.Identifiers[i:i+1], decl.Type, decl.Values[i:i+1]))
			}
		case *ast.Var:
			if len(decl.Lhs) == len(decl.Rhs) {
				for i := range decl.Lhs {
					vars = append(vars, ast.NewVar(decl.Pos(), decl.Lhs[i:i+1], decl.Type, decl.Rhs[i:i+1]))
				}
			} else {
				vars = append(vars, decl)
			}
		}
	}

	// Removes from deps all non-global dependencies.
	for decl, ds := range deps {
		newDs := []*ast.Identifier{}
		for _, d := range ds {
			// Is d a global identifier?
			for _, c := range consts {
				if c.Identifiers[0].Name == d.Name {
					newDs = append(newDs, d)
				}
			}
			for _, f := range funcs {
				if f.Ident.Name == d.Name {
					newDs = append(newDs, d)
				}
			}
			for _, v := range vars {
				for _, left := range v.Lhs {
					if left.Name == d.Name {
						newDs = append(newDs, d)
					}
				}
			}
		}
		deps[decl] = newDs
	}

	err := detectConstantsLoop(consts, deps)
	if err != nil {
		return err
	}
	err = detectVarsLoop(vars, deps)
	if err != nil {
		return err
	}

	// Sorts constants.
	sortedConsts := []*ast.Const{}
constsLoop:
	for len(consts) > 0 {
		loop := true
		// Searches for next constant with resolved deps.
		for i, c := range consts {
			depsOk := true
			for _, dep := range deps[c.Identifiers[0]] {
				found := false
				for _, resolvedC := range sortedConsts {
					if dep.Name == resolvedC.Identifiers[0].Name {
						// This dependency has been resolved: move
						// on checking for next one.
						found = true
						break
					}
				}
				if !found {
					// A dependency of c cannot be found.
					depsOk = false
					break
				}
			}
			if depsOk {
				// All dependencies of consts are resolved.
				sortedConsts = append(sortedConsts, c)
				consts = append(consts[:i], consts[i+1:]...)
				continue constsLoop
			}
		}
		if loop {
			// All remaining constants are added as "sorted"; a later
			// stage of typechecking will report the error (usually a not
			// defined symbol).
			sortedConsts = append(sortedConsts, consts...)
		}
	}

	// Sorts variables.
	sortedVars := []*ast.Var{}
varsLoop:
	for len(vars) > 0 {
		loop := true
		// Searches for next variable with resolved deps.
		for i, v := range vars {
			depsOk := true
			for _, dep := range deps[v.Lhs[0]] {
				found := false
			resolvedLoop:
				for _, resolvedV := range sortedVars {
					for _, left := range resolvedV.Lhs {
						if dep.Name == left.Name {
							// This dependency has been resolved: move
							// on checking for next one.
							found = true
							break resolvedLoop
						}
					}
				}
				for _, resolvedC := range sortedConsts {
					if dep.Name == resolvedC.Identifiers[0].Name {
						// This dependency has been resolved: move
						// on checking for next one.
						found = true
						break
					}
				}
				for _, f := range funcs {
					if dep.Name == f.Ident.Name {
						// This dependency has been resolved: move
						// on checking for next one.
						found = true
						break
					}
				}
				if !found {
					// A dependency of v cannot be found.
					depsOk = false
					break
				}
			}
			if depsOk {
				// All dependencies of vars are resolved.
				sortedVars = append(sortedVars, v)
				vars = append(vars[:i], vars[i+1:]...)
				continue varsLoop
			}
		}
		if loop {
			// All remaining vars are added as "sorted"; a later stage of
			// typechecking will report the error (usually a not defined
			// symbol).
			sortedVars = append(sortedVars, vars...)
		}
	}

	sorted := []ast.Node{}
	for _, imp := range imports {
		sorted = append(sorted, imp)
	}
	for _, c := range sortedConsts {
		sorted = append(sorted, c)
	}
	for _, v := range sortedVars {
		sorted = append(sorted, v)
	}
	for _, f := range funcs {
		sorted = append(sorted, f)
	}
	pkg.Declarations = sorted

	return nil

}

// CheckPackage type checks a package.
func CheckPackage(tree *ast.Tree, deps GlobalsDependencies, imports map[string]*native.GoPackage, pkgInfos map[string]*PackageInfo) (err error) {

	defer func() {
		if r := recover(); r != nil {
			if rerr, ok := r.(*Error); ok {
				err = rerr
			} else {
				panic(r)
			}
		}
	}()

	if len(tree.Nodes) == 0 {
		return errors.New("expected 'package', found EOF")
	}
	packageNode, ok := tree.Nodes[0].(*ast.Package)
	if !ok {
		t := fmt.Sprintf("%T", tree.Nodes[0])
		t = strings.ToLower(t[len("*ast."):])
		return fmt.Errorf("expected 'package', found '%s'", t)
	}

	tc := NewTypechecker(tree.Path, false)
	tc.Universe = Universe

	err = sortDeclarations(packageNode, deps)
	if err != nil {
		return err
	}

	// Defines functions in file/package block before checking all
	// declarations.
	for _, d := range packageNode.Declarations {
		if f, ok := d.(*ast.Func); ok {
			if f.Ident.Name == "init" || f.Ident.Name == "main" {
				if len(f.Type.Parameters) > 0 || len(f.Type.Result) > 0 {
					panic(tc.errorf(f.Ident, "func %s must have no arguments and no return values", f.Ident.Name))
				}
			}
			if f.Ident.Name != "init" {
				if _, ok := tc.filePackageBlock[f.Ident.Name]; ok {
					panic(tc.errorf(f.Ident, "%s redeclared in this block", f.Ident.Name))
				}
				tc.filePackageBlock[f.Ident.Name] = scopeElement{t: &TypeInfo{Type: tc.typeof(f.Type, noEllipses).Type}}
			}
		}
	}

	for _, d := range packageNode.Declarations {
		switch d := d.(type) {
		case *ast.Import:
			importedPkg := &PackageInfo{}
			if d.Tree == nil {
				// Go package.
				goPkg, ok := imports[d.Path]
				if !ok {
					return tc.errorf(d, "cannot find package %q", d.Path)
				}
				importedPkg.Declarations = make(map[string]*TypeInfo, len(goPkg.Declarations))
				for n, d := range ToTypeCheckerScope(goPkg) {
					importedPkg.Declarations[n] = d.t
				}
				importedPkg.Name = goPkg.Name
			} else {
				// Scrigo package.
				var err error
				err = CheckPackage(d.Tree, nil, nil, pkgInfos) // TODO(Gianluca): where are deps?
				importedPkg = pkgInfos[d.Tree.Path]
				if err != nil {
					return err
				}
			}
			if d.Ident == nil {
				tc.filePackageBlock[importedPkg.Name] = scopeElement{t: &TypeInfo{Value: importedPkg, Properties: PropertyIsPackage}}
				tc.unusedImports[importedPkg.Name] = nil
			} else {
				switch d.Ident.Name {
				case "_":
				case ".":
					tc.unusedImports[importedPkg.Name] = nil
					for ident, ti := range importedPkg.Declarations {
						tc.unusedImports[importedPkg.Name] = append(tc.unusedImports[importedPkg.Name], ident)
						tc.filePackageBlock[ident] = scopeElement{t: ti}
					}
				default:
					tc.filePackageBlock[d.Ident.Name] = scopeElement{t: &TypeInfo{Value: importedPkg, Properties: PropertyIsPackage}}
					tc.unusedImports[d.Ident.Name] = nil
				}
			}
		case *ast.Func:
			tc.addScope()
			tc.ancestors = append(tc.ancestors, &ancestor{len(tc.Scopes), d})
			// Adds parameters to the function body scope.
			fillParametersTypes(d.Type.Parameters)
			isVariadic := d.Type.IsVariadic
			for i, param := range d.Type.Parameters {
				t := tc.checkType(param.Type, noEllipses)
				new := ast.NewValue(t.Type)
				tc.replaceTypeInfo(param.Type, new)
				param.Type = new
				if param.Ident != nil {
					if isVariadic && i == len(d.Type.Parameters)-1 {
						tc.assignScope(param.Ident.Name, &TypeInfo{Type: reflect.SliceOf(t.Type), Properties: PropertyAddressable}, nil)
					} else {
						tc.assignScope(param.Ident.Name, &TypeInfo{Type: t.Type, Properties: PropertyAddressable}, nil)
					}
				}
			}
			// Adds named return values to the function body scope.
			fillParametersTypes(d.Type.Result)
			for _, ret := range d.Type.Result {
				t := tc.checkType(ret.Type, noEllipses)
				new := ast.NewValue(t.Type)
				tc.replaceTypeInfo(ret.Type, new)
				ret.Type = new
				if ret.Ident != nil {
					tc.assignScope(ret.Ident.Name, &TypeInfo{Type: t.Type, Properties: PropertyAddressable}, nil)
				}
			}
			tc.checkNodes(d.Body.Nodes)
			tc.ancestors = tc.ancestors[:len(tc.ancestors)-1]
			tc.removeCurrentScope()
		case *ast.Const:
			tc.checkAssignment(d)
		case *ast.Var:
			tc.checkAssignment(d)
		}
	}

	for pkg := range tc.unusedImports {
		return tc.errorf(new(ast.Position), "imported and not used: \"%s\"", pkg) // TODO (Gianluca): position is not correct.
	}

	if packageNode.Name == "main" {
		main, ok := tc.filePackageBlock["main"]
		if !ok {
			return tc.errorf(new(ast.Position), "function main is undeclared in the main package")
		}
		if main.t.Type.Kind() != reflect.Func || main.t.Addressable() {
			return tc.errorf(new(ast.Position), "cannot declare main - must be func")
		}
	}

	pkgInfo := &PackageInfo{
		Name:         packageNode.Name,
		Declarations: make(map[string]*TypeInfo, len(packageNode.Declarations)),
		TypeInfo:     tc.TypeInfo,
	}
	pkgInfo.Declarations = make(map[string]*TypeInfo)
	for ident, ti := range tc.filePackageBlock {
		pkgInfo.Declarations[ident] = ti.t
	}
	pkgInfo.IndirectVars = tc.IndirectVars

	if pkgInfos == nil {
		pkgInfos = make(map[string]*PackageInfo)
	}
	pkgInfos[tree.Path] = pkgInfo

	return nil
}
