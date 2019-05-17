package compiler

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"scrigo/internal/compiler/ast"
)

// TODO (Gianluca): find a better name.
// TODO (Gianluca): this identifier must be accessed from outside as
// "scrigo.Package" or something similar.
type GoPackage struct {
	Name         string
	Declarations map[string]interface{}
}

func (gp *GoPackage) ToTypeCheckerScope() TypeCheckerScope {
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

// notCheckedGlobal represents the type info of a not type-checked package
// declaration.
var notCheckedGlobal = &TypeInfo{}

// CheckPackage type checks a package.
func CheckPackage(tree *ast.Tree, imports map[string]*GoPackage, pkgInfos map[string]*PackageInfo) (err error) {

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

	for _, n := range packageNode.Declarations {
		switch n := n.(type) {
		case *ast.Import:
			importedPkg := &PackageInfo{}
			if n.Tree == nil {
				// Go package.
				goPkg, ok := imports[n.Path]
				if !ok {
					return tc.errorf(n, "cannot find package %q", n.Path)
				}
				importedPkg.Declarations = make(map[string]*TypeInfo, len(goPkg.Declarations))
				for n, d := range goPkg.ToTypeCheckerScope() {
					importedPkg.Declarations[n] = d.t
				}
				importedPkg.Name = goPkg.Name
			} else {
				// Scrigo package.
				var err error
				err = CheckPackage(n.Tree, nil, pkgInfos)
				importedPkg = pkgInfos[n.Tree.Path]
				if err != nil {
					return err
				}
			}
			if n.Ident == nil {
				tc.filePackageBlock[importedPkg.Name] = scopeElement{t: &TypeInfo{Value: importedPkg, Properties: PropertyIsPackage}}
				tc.unusedImports[importedPkg.Name] = nil
			} else {
				switch n.Ident.Name {
				case "_":
				case ".":
					tc.unusedImports[importedPkg.Name] = nil
					for ident, ti := range importedPkg.Declarations {
						tc.unusedImports[importedPkg.Name] = append(tc.unusedImports[importedPkg.Name], ident)
						tc.filePackageBlock[ident] = scopeElement{t: ti}
					}
				default:
					tc.filePackageBlock[n.Ident.Name] = scopeElement{t: &TypeInfo{Value: importedPkg, Properties: PropertyIsPackage}}
					tc.unusedImports[n.Ident.Name] = nil
				}
			}
		case *ast.Const:
			for i := range n.Identifiers {
				name := n.Identifiers[i].Name
				if _, ok := tc.filePackageBlock[name]; ok {
					panic(tc.errorf(n.Identifiers[i], "%s redeclared in this block", name))
				}
				tc.filePackageBlock[name] = scopeElement{t: notCheckedGlobal}
				tc.declarations = append(tc.declarations, &Declaration{Node: n, Ident: name, Value: n.Values[i], Type: n.Type, DeclType: DeclConst})
			}
		case *ast.Var:
			if len(n.Values) == 0 {
				typ := tc.checkType(n.Type, noEllipses)
				// Replaces the type node with a value holding a reflect.Type.
				n.Values = make([]ast.Expression, len(n.Identifiers))
				var zero interface{}
				if typ.Type.Kind() == reflect.Interface {
					zero = nil
				} else {
					zero = reflect.Zero(typ.Type).Interface()
				}
				for i := range n.Identifiers {
					n.Values[i] = ast.NewValue(zero)
					tc.TypeInfo[n.Values[i]] = &TypeInfo{Type: typ.Type}
				}
				return
			}
			for i := range n.Identifiers {
				name := n.Identifiers[i].Name
				if _, ok := tc.filePackageBlock[name]; ok {
					panic(tc.errorf(n.Identifiers[i], "%s redeclared in this block", name))
				}
				tc.declarations = append(tc.declarations, &Declaration{Node: n, Ident: name, Value: n.Values[i], Type: n.Type, DeclType: DeclVar, Identifier: n.Identifiers[0]}) // TODO (Gianluca): add support for var a, b, c = f()
				tc.filePackageBlock[name] = scopeElement{t: notCheckedGlobal}
			}
		case *ast.TypeDeclaration:
			// TODO (Gianluca): add support for types referring to other
			// types defined later. See
			// https://play.golang.org/p/RJ8WruPku0U.
			// TODO (Gianluca): all types are defined as alias
			// declarations.
			if isBlankIdentifier(n.Identifier) {
				continue
			}
			name := n.Identifier.Name
			typ := tc.checkType(n.Type, noEllipses)
			tc.filePackageBlock[name] = scopeElement{t: typ}
		case *ast.Func:
			if n.Ident.Name == "init" {
				if len(n.Type.Parameters) > 0 || len(n.Type.Result) > 0 {
					panic(tc.errorf(n.Ident, "func init must have no arguments and no return values"))
				}
			} else {
				if _, ok := tc.filePackageBlock[n.Ident.Name]; ok {
					panic(tc.errorf(n.Ident, "%s redeclared in this block", n.Ident.Name))
				}
			}
			tc.declarations = append(tc.declarations, &Declaration{Node: n, Ident: n.Ident.Name, Value: n.Body, Type: n.Type, DeclType: DeclFunc})
			tc.filePackageBlock[n.Ident.Name] = scopeElement{t: notCheckedGlobal}
		}
	}

	// Constants.
	for _, c := range tc.declarations {
		if c.DeclType == DeclConst {
			tc.currentGlobal = c.Ident
			tc.globalEvalPath = []string{c.Ident}
			tc.globalTemp = make(map[string]*TypeInfo)
			ti := tc.checkExpression(c.Value.(ast.Expression))
			if !ti.IsConstant() {
				return tc.errorf(c.Value, "const initializer %v is not a constant", c.Value)
			}
			if c.Type != nil {
				typ := tc.checkType(c.Type, noEllipses)
				if !isAssignableTo(ti, typ.Type) {
					return tc.errorf(c.Value, "cannot convert %v (type %s) to type %v", c.Value, ti.String(), typ.Type)
				}
			}
			tc.filePackageBlock[c.Ident] = scopeElement{t: ti}
		}
	}

	// Functions.
	for _, v := range tc.declarations {
		if v.DeclType == DeclFunc {
			tc.addScope()
			tc.ancestors = append(tc.ancestors, &ancestor{len(tc.Scopes), v.Node})
			// Adds parameters to the function body scope.
			fillParametersTypes(v.Type.(*ast.FuncType).Parameters)
			isVariadic := v.Type.(*ast.FuncType).IsVariadic
			for i, param := range v.Type.(*ast.FuncType).Parameters {
				t := tc.checkType(param.Type, noEllipses)
				new := ast.NewValue(t.Type)
				tc.replaceTypeInfo(param.Type, new)
				param.Type = new
				if param.Ident != nil {
					if isVariadic && i == len(v.Type.(*ast.FuncType).Parameters)-1 {
						tc.assignScope(param.Ident.Name, &TypeInfo{Type: reflect.SliceOf(t.Type), Properties: PropertyAddressable}, nil)
					} else {
						tc.assignScope(param.Ident.Name, &TypeInfo{Type: t.Type, Properties: PropertyAddressable}, nil)
					}
				}
			}
			// Adds named return values to the function body scope.
			fillParametersTypes(v.Type.(*ast.FuncType).Result)
			for _, ret := range v.Type.(*ast.FuncType).Result {
				t := tc.checkType(ret.Type, noEllipses)
				new := ast.NewValue(t.Type)
				tc.replaceTypeInfo(ret.Type, new)
				ret.Type = new
				if ret.Ident != nil {
					tc.assignScope(ret.Ident.Name, &TypeInfo{Type: t.Type, Properties: PropertyAddressable}, nil)
				}
			}
			tc.currentGlobal = v.Ident
			tc.globalEvalPath = []string{v.Ident}
			tc.globalTemp = make(map[string]*TypeInfo)
			tc.filePackageBlock[v.Ident] = scopeElement{t: &TypeInfo{Type: tc.typeof(v.Type, noEllipses).Type}}
			tc.checkNodes(v.Value.(*ast.Block).Nodes)
			tc.initOrder = append(tc.initOrder, v.Ident)
			tc.ancestors = tc.ancestors[:len(tc.ancestors)-1]
			tc.removeCurrentScope()
		}
	}

	// Variables.
	for unresolvedDeps := true; unresolvedDeps; {
		unresolvedDeps = false
		for _, v := range tc.declarations {
			if v.DeclType == DeclVar {
				tc.IndirectVars[v.Identifier] = true
				tc.currentGlobal = v.Ident
				tc.globalEvalPath = []string{v.Ident}
				tc.globalTemp = make(map[string]*TypeInfo)
				ti := tc.checkExpression(v.Value.(ast.Expression))
				if v.Type != nil {
					typ := tc.checkType(v.Type, noEllipses)
					if !isAssignableTo(ti, typ.Type) {
						return tc.errorf(v.Value, "cannot convert %v (type %s) to type %v", v.Value, ti.String(), typ.Type)
					}
				}
				varTi := &TypeInfo{Type: ti.Type, Properties: PropertyAddressable}
				tc.filePackageBlock[v.Ident] = scopeElement{t: varTi}
				if !tc.tryAddingToInitOrder(v.Ident) {
					unresolvedDeps = true
				}
				tc.TypeInfo[v.Identifier] = varTi
				// Replaces value's node and typeinfo if it's a constant.
				if ti.IsConstant() {
					for i, ident := range v.Node.(*ast.Var).Identifiers {
						if ident.Name == v.Ident {
							new := ast.NewValue(typedValue(ti, varTi.Type))
							tc.replaceTypeInfo(v.Node.(*ast.Var).Values[i], new)
							v.Node.(*ast.Var).Values[i] = new
						}
					}
				}
			}
		}
	}
	tc.globalTemp = nil
	tc.globalEvalPath = nil
	tc.currentGlobal = ""

	for pkg := range tc.unusedImports {
		// TODO (Gianluca): position is not correct.
		return tc.errorf(new(ast.Position), "imported and not used: \"%s\"", pkg)
	}

	// Checks if main is defined and if it's a function.
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

	// Sort variables.
	// TODO (Gianluca): if a variable declaration is already
	// internal-ordered, there's no need to split it in many
	// single-variable declarations, just put it in orderedVars as is.
	orderedVars := []ast.Node{}
OrderedVarsLoop:
	for _, v := range tc.initOrder {
		for _, n := range packageNode.Declarations {
			switch n := n.(type) {
			case *ast.Var:
				for i, ident := range n.Identifiers {
					if ident.Name == v {
						if len(n.Identifiers) == len(n.Values) {
							assignment := ast.NewVar(n.Pos(), []*ast.Identifier{ident}, n.Type, []ast.Expression{n.Values[i]})
							orderedVars = append(orderedVars, assignment)
							continue OrderedVarsLoop
						} else {
							orderedVars = append(orderedVars, n)
							continue OrderedVarsLoop
						}
					}
				}
			}
		}
	}
	// Init functions.
	initNodes := []ast.Node{}
	for i, n := range packageNode.Declarations {
		f, ok := n.(*ast.Func)
		if ok {
			if f.Ident.Name == "init" {
				initNodes = append(initNodes, f)
				packageNode.Declarations[i] = nil
			}
		}
	}
	// Imports and functions.
	funcNodes := []ast.Node{}
	importNodes := []ast.Node{}
	for _, n := range packageNode.Declarations {
		switch n := n.(type) {
		case nil:
		case *ast.Var:
		case *ast.Import:
			importNodes = append(importNodes, n)
		case *ast.Func:
			funcNodes = append(funcNodes, n)
		}
	}

	orderedNodes := []ast.Node{}
	orderedNodes = append(orderedNodes, importNodes...)
	orderedNodes = append(orderedNodes, funcNodes...)
	orderedNodes = append(orderedNodes, orderedVars...)
	orderedNodes = append(orderedNodes, initNodes...)
	packageNode.Declarations = orderedNodes

	if pkgInfos == nil {
		pkgInfos = make(map[string]*PackageInfo)
	}
	pkgInfos[tree.Path] = pkgInfo

	return nil
}

// tryAddingToInitOrder tries to add name to the initialization order. Returns
// true on success.
func (tc *typechecker) tryAddingToInitOrder(name string) bool {
	for _, dep := range tc.varDeps[name] {
		if !sliceContainsString(tc.initOrder, dep) {
			return false
		}
	}
	if !sliceContainsString(tc.initOrder, name) {
		tc.initOrder = append(tc.initOrder, name)
	}
	return true
}
