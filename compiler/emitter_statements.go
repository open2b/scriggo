// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"path/filepath"
	"reflect"

	"github.com/open2b/scriggo/compiler/ast"
	"github.com/open2b/scriggo/runtime"
)

// emitNodes emits instructions for nodes.
func (em *emitter) emitNodes(nodes []ast.Node) {

	for _, node := range nodes {
		switch node := node.(type) {

		case *ast.Assignment:
			em.emitAssignmentNode(node)

		case *ast.Block:
			em.fb.enterScope()
			em.emitNodes(node.Nodes)
			em.fb.exitScope()

		case *ast.Break:
			if em.breakable {
				if em.breakLabel == nil {
					label := em.fb.newLabel()
					em.breakLabel = &label
				}
				em.fb.emitGoto(*em.breakLabel)
			} else {
				if node.Label != nil {
					panic("TODO(Gianluca): not implemented")
				}
				em.fb.emitBreak(em.rangeLabels[len(em.rangeLabels)-1][0])
				em.fb.emitGoto(em.rangeLabels[len(em.rangeLabels)-1][1])
			}

		case *ast.Comment:
			// Nothing to do.

		case *ast.Const:
			// Nothing to do.

		case *ast.Continue:
			if node.Label != nil {
				panic("TODO(Gianluca): not implemented")
			}
			forHead := em.rangeLabels[len(em.rangeLabels)-1][0]
			if em.inForRange {
				em.fb.emitContinue(forHead)
			} else {
				em.fb.emitGoto(forHead)
			}

		case *ast.Defer:
			call := node.Call.(*ast.Call)
			if em.ti(call.Func) == showMacroIgnoredTi {
				// Nothing to do
				continue
			}
			if em.builtinCallName(call) == "recover" {
				stackShift := em.fb.currentStackShift()
				backup := em.fb
				fnReg := em.fb.newRegister(reflect.Func)
				fn := em.fb.emitFunc(fnReg, reflect.FuncOf(nil, nil, false))
				em.fb = newBuilder(fn, em.fb.getPath())
				em.fb.emitRecover(0, true)
				em.fb.emitReturn()
				em.fb = backup
				em.fb.emitDefer(fnReg, 0, stackShift, runtime.StackShift{0, 0, 0, 0}, fn.Type)
				continue
			}
			em.fb.enterStack()
			_, _ = em.emitCallNode(call, false, true)
			em.fb.exitStack()

		case *ast.Import:
			if em.isTemplate {
				// Import a template file.
				// Precompiled packages have been already handled by the type
				// checker and should be ignored by the emitter.
				if ext := filepath.Ext(node.Path); ext != "" {
					inits := em.emitImport(node, true)
					if len(inits) > 0 && !em.alreadyInitializedTemplatePkgs[node.Tree.Path] {
						for _, initFunc := range inits {
							index := em.fb.addFunction(initFunc)
							em.fb.emitCall(int8(index), runtime.StackShift{}, nil)
						}
						em.alreadyInitializedTemplatePkgs[node.Path] = true
					}
				}
			}

		case *ast.Fallthrough:
			// Nothing to do: fallthrough nodes are handled by method
			// emitter.emitSwitch.

		case *ast.For:
			currentBreakable := em.breakable
			currentBreakLabel := em.breakLabel
			em.breakable = true
			em.breakLabel = nil
			em.fb.enterScope()
			if node.Init != nil {
				em.emitNodes([]ast.Node{node.Init})
			}
			if node.Condition != nil {
				forHead := em.fb.newLabel()
				forPost := em.fb.newLabel()
				em.fb.setLabelAddr(forHead)
				em.emitCondition(node.Condition)
				endForLabel := em.fb.newLabel()
				em.fb.emitGoto(endForLabel)
				em.rangeLabels = append(em.rangeLabels, [2]label{forPost, endForLabel})
				em.emitNodes(node.Body)
				em.rangeLabels = em.rangeLabels[:len(em.rangeLabels)-1]
				em.fb.setLabelAddr(forPost)
				if node.Post != nil {
					em.emitNodes([]ast.Node{node.Post})
				}
				em.fb.emitGoto(forHead)
				em.fb.setLabelAddr(endForLabel)
			} else {
				forLabel := em.fb.newLabel()
				em.fb.setLabelAddr(forLabel)
				endForLabel := em.fb.newLabel()
				em.rangeLabels = append(em.rangeLabels, [2]label{forLabel, endForLabel})
				em.emitNodes(node.Body)
				if node.Post != nil {
					em.emitNodes([]ast.Node{node.Post})
				}
				em.fb.emitGoto(forLabel)
				em.fb.setLabelAddr(endForLabel)
			}
			em.fb.exitScope()
			if em.breakLabel != nil {
				em.fb.setLabelAddr(*em.breakLabel)
			}
			em.breakable = currentBreakable
			em.breakLabel = currentBreakLabel

		case *ast.ForRange:
			inForRange := em.inForRange
			em.inForRange = true
			em.fb.enterScope()
			vars := node.Assignment.Lhs
			expr := node.Assignment.Rhs[0]
			exprType := em.typ(expr)
			exprReg, kExpr := em.emitExprK(expr, exprType)
			if exprType.Kind() != reflect.String && kExpr {
				kExpr = false
				exprReg = em.emitExpr(expr, exprType)
			}
			indexReg := int8(0)
			if len(vars) >= 1 && !isBlankIdentifier(vars[0]) {
				name := vars[0].(*ast.Identifier).Name
				if node.Assignment.Type == ast.AssignmentDeclaration {
					indexReg = em.fb.newRegister(reflect.Int)
					em.fb.bindVarReg(name, indexReg)
				} else {
					indexReg = em.fb.scopeLookup(name)
				}
			}
			elem := int8(0)
			if len(vars) == 2 && !isBlankIdentifier(vars[1]) {
				name := vars[1].(*ast.Identifier).Name
				if node.Assignment.Type == ast.AssignmentDeclaration {
					elem = em.fb.newRegister(em.typ(vars[1]).Kind())
					em.fb.bindVarReg(name, elem)
				} else {
					elem = em.fb.scopeLookup(name)
				}
			}
			rangeLabel := em.fb.newLabel()
			em.fb.setLabelAddr(rangeLabel)
			endRange := em.fb.newLabel()
			em.rangeLabels = append(em.rangeLabels, [2]label{rangeLabel, endRange})
			em.fb.emitRange(kExpr, exprReg, indexReg, elem, exprType.Kind())
			em.fb.emitGoto(endRange)
			em.fb.enterScope()
			em.emitNodes(node.Body)
			em.fb.emitContinue(rangeLabel)
			em.fb.setLabelAddr(endRange)
			em.rangeLabels = em.rangeLabels[:len(em.rangeLabels)-1]
			em.fb.exitScope()
			em.fb.exitScope()
			em.inForRange = inForRange

		case *ast.Go:
			call := node.Call.(*ast.Call)
			if em.ti(call.Func) == showMacroIgnoredTi {
				// Nothing to do
				continue
			}
			em.fb.enterStack()
			_, _ = em.emitCallNode(call, true, false)
			em.fb.exitStack()

		case *ast.Goto:
			if lab, ok := em.labels[em.fb.fn][node.Label.Name]; ok {
				em.fb.emitGoto(lab)
			} else {
				if em.labels[em.fb.fn] == nil {
					em.labels[em.fb.fn] = make(map[string]label)
				}
				lab = em.fb.newLabel()
				em.fb.emitGoto(lab)
				em.labels[em.fb.fn][node.Label.Name] = lab
			}

		case *ast.If:
			em.fb.enterScope()
			if node.Init != nil {
				em.emitNodes([]ast.Node{node.Init})
			}
			em.emitCondition(node.Condition)
			if node.Else == nil {
				endIfLabel := em.fb.newLabel()
				em.fb.emitGoto(endIfLabel)
				em.fb.enterScope()
				em.emitNodes(node.Then.Nodes)
				em.fb.exitScope()
				em.fb.setLabelAddr(endIfLabel)
			} else {
				elseLabel := em.fb.newLabel()
				em.fb.emitGoto(elseLabel)
				em.fb.enterScope()
				em.emitNodes(node.Then.Nodes)
				em.fb.exitScope()
				endIfLabel := em.fb.newLabel()
				em.fb.emitGoto(endIfLabel)
				em.fb.setLabelAddr(elseLabel)
				switch els := node.Else.(type) {
				case *ast.If:
					em.emitNodes([]ast.Node{els})
				case *ast.Block:
					em.emitNodes(els.Nodes)
				}
				em.fb.setLabelAddr(endIfLabel)
			}
			em.fb.exitScope()

		case *ast.ShowPartial:
			// The scope that shows the partial file must be hidden from such
			// file, otherwise the shown file would see the variables in the
			// scope that shows it instead of global variables (note that the
			// emitter gives precedence to local variables respect to global
			// variables).
			path := em.fb.getPath()
			em.fb.changePath(node.Tree.Path)
			scopes := em.fb.scopes
			em.fb.scopes = nil
			em.fb.enterScope()
			em.emitNodes(node.Tree.Nodes)
			em.fb.exitScope()
			em.fb.changePath(path)
			em.fb.scopes = scopes

		case *ast.Label:
			if _, found := em.labels[em.fb.fn][node.Ident.Name]; !found {
				if em.labels[em.fb.fn] == nil {
					em.labels[em.fb.fn] = make(map[string]label)
				}
				em.labels[em.fb.fn][node.Ident.Name] = em.fb.newLabel()
			}
			em.fb.setLabelAddr(em.labels[em.fb.fn][node.Ident.Name])
			if node.Statement != nil {
				em.emitNodes([]ast.Node{node.Statement})
			}

		case *ast.Return:
			offset := [4]int8{}
			// Emit return statements with a function call that returns more
			// than one value.
			//
			// Example:
			//
			//	return fmt.Println("text")
			//
			fnType := em.fb.fn.Type
			if len(node.Values) == 1 && fnType.NumOut() > 1 {
				returnedRegs, types := em.emitCallNode(node.Values[0].(*ast.Call), false, false)
				for i, typ := range types {
					var dstReg int8
					switch kindToType(typ.Kind()) {
					case intRegister:
						offset[0]++
						dstReg = offset[0]
					case floatRegister:
						offset[1]++
						dstReg = offset[1]
					case stringRegister:
						offset[2]++
						dstReg = offset[2]
					case generalRegister:
						offset[3]++
						dstReg = offset[3]
					}
					em.changeRegister(false, returnedRegs[i], dstReg, typ, fnType.Out(i))
				}
				continue
			}
			for i, v := range node.Values {
				typ := fnType.Out(i)
				var reg int8
				switch kindToType(typ.Kind()) {
				case intRegister:
					offset[0]++
					reg = offset[0]
				case floatRegister:
					offset[1]++
					reg = offset[1]
				case stringRegister:
					offset[2]++
					reg = offset[2]
				case generalRegister:
					offset[3]++
					reg = offset[3]
				}
				em.emitExprR(v, typ, reg)
			}
			em.fb.emitReturn()

		case *ast.Select:
			currentBreakable := em.breakable
			currentBreakLabel := em.breakLabel
			em.breakable = true
			em.breakLabel = nil
			em.emitSelect(node)
			em.breakable = currentBreakable
			em.breakLabel = currentBreakLabel

		case *ast.Send:
			chanType := em.typ(node.Channel)
			chann := em.emitExpr(node.Channel, chanType)
			value := em.emitExpr(node.Value, chanType.Elem())
			em.fb.emitSend(chann, value, node.Pos(), chanType.Elem().Kind())

		case *ast.Show:
			// render([implicit *vm.Env,] gD io.Writer, gE interface{}, iA ast.Context)
			em.emitExprR(node.Expr, emptyInterfaceType, em.fb.templateRegs.gE)
			em.fb.emitMove(true, int8(node.Context), em.fb.templateRegs.iA, reflect.Int, false)
			if em.inURL {
				// In a URL context: use the urlWriter, that implements io.Writer.
				em.fb.emitMove(false, em.fb.templateRegs.gF, em.fb.templateRegs.gD, reflect.Interface, false)
			} else {
				// Not in a URL context: use the default writer.
				em.fb.emitMove(false, em.fb.templateRegs.gA, em.fb.templateRegs.gD, reflect.Interface, false)
			}
			shift := runtime.StackShift{em.fb.templateRegs.iA - 1, 0, 0, em.fb.templateRegs.gC}
			em.fb.emitCallIndirect(em.fb.templateRegs.gC, 0, shift, node.Pos(), renderFuncType)

		case *ast.Switch:
			currentBreakable := em.breakable
			currentBreakLabel := em.breakLabel
			em.breakable = true
			em.breakLabel = nil
			em.emitSwitch(node)
			if em.breakLabel != nil {
				em.fb.setLabelAddr(*em.breakLabel)
			}
			em.breakable = currentBreakable
			em.breakLabel = currentBreakLabel

		case *ast.Text:
			// Write(gE []byte) (iA int, gD error)
			index := len(em.fb.fn.Data)
			data := node.Text[node.Cut.Left : len(node.Text)-node.Cut.Right]
			if len(data) != 0 {
				em.fb.fn.Data = append(em.fb.fn.Data, data)
				em.fb.emitLoadData(int16(index), em.fb.templateRegs.gE)
				var writeFun int8
				if em.inURL {
					// In a URL context: getting the method WriteText of an the
					// urlWriter, that has the same sign of the method Write which
					// implements interface io.Writer.
					em.fb.enterStack()
					writeFun = em.fb.newRegister(reflect.Func)
					em.fb.emitMethodValue("WriteText", em.fb.templateRegs.gF, writeFun, node.Pos())
					em.fb.exitStack()
				} else {
					writeFun = em.fb.templateRegs.gB
				}
				em.fb.emitCallIndirect(
					writeFun, // register
					0,        // numVariadic
					runtime.StackShift{em.fb.templateRegs.iA - 1, 0, 0, em.fb.templateRegs.gC},
					node.Pos(),
					ioWriterWriteType, // functionType
				)
			}

		case *ast.TypeDeclaration:
			// Nothing to do.

		case *ast.TypeSwitch:
			currentBreakable := em.breakable
			currentBreakLabel := em.breakLabel
			em.breakable = true
			em.breakLabel = nil
			em.emitTypeSwitch(node)
			if em.breakLabel != nil {
				em.fb.setLabelAddr(*em.breakLabel)
			}
			em.breakable = currentBreakable
			em.breakLabel = currentBreakLabel

		case *ast.URL:
			// Entering inside an URL context; this will affect the way that
			// values and text are rendered.
			em.inURL = true
			// Call method Reset of urlWriter.
			em.fb.enterStack()
			method := em.fb.newRegister(reflect.Func)
			em.fb.emitMethodValue("StartURL", em.fb.templateRegs.gF, method, node.Pos())
			ss := em.fb.currentStackShift()
			quoteArg := em.fb.newRegister(reflect.Bool)
			isSetArg := em.fb.newRegister(reflect.Bool)
			var quote, isSet int8
			if node.Context == ast.ContextAttribute {
				quote = 1
			}
			if node.Attribute == "srcset" {
				isSet = 1
			}
			em.changeRegister(true, quote, quoteArg, boolType, boolType)
			em.changeRegister(true, isSet, isSetArg, boolType, boolType)
			em.fb.emitCallIndirect(method, 0, ss, node.Pos(), urlEscaperStartURLType)
			em.fb.exitStack()
			// Emit the nodes in the URL.
			em.emitNodes(node.Value)
			// Exiting from an URL context.
			em.inURL = false

		case *ast.Var:
			addresses := make([]address, len(node.Lhs))
			// Variable names must be bind to the corresponding register after
			// emitting the code that evaluates the expression, otherwise the
			// declaration of a variable on the left side of = would shadow a
			// variable with the same name on the right (they are two different
			// variables).
			varsToBind := make(map[string]int8, len(node.Lhs))
			for i, v := range node.Lhs {
				if isBlankIdentifier(v) {
					addresses[i] = em.addressBlankIdent(v.Pos())
				} else {
					staticType := em.typ(v)
					var varr int8
					if em.varStore.mustBeDeclaredAsIndirect(v) {
						varr = em.fb.newIndirectRegister()
						addresses[i] = em.addressNewIndirectVar(varr, staticType, v.Pos(), 0)
					} else {
						varr = em.fb.newRegister(staticType.Kind())
						addresses[i] = em.addressLocalVar(varr, staticType, v.Pos(), 0)
					}
					varsToBind[v.Name] = varr
				}
			}
			em.assignValuesToAddresses(addresses, node.Rhs)
			for name, reg := range varsToBind {
				em.fb.bindVarReg(name, reg)
			}

		case ast.Expression:
			em.fb.enterStack()
			em.emitExprR(node, reflect.Type(nil), 0)
			em.fb.exitStack()

		default:
			panic(fmt.Sprintf("BUG: node %T not supported", node)) // remove.

		}

	}

}

// emitAssignmentNode emits the instructions for an assignment node.
func (em *emitter) emitAssignmentNode(node *ast.Assignment) {

	// Emit a short declaration.
	if node.Type == ast.AssignmentDeclaration {
		addresses := make([]address, len(node.Lhs))
		for i, v := range node.Lhs {
			pos := v.Pos()
			if isBlankIdentifier(v) {
				addresses[i] = em.addressBlankIdent(pos)
				continue
			}
			v := v.(*ast.Identifier)
			varType := em.typ(v)
			// Declare an indirect local variable.
			if em.varStore.mustBeDeclaredAsIndirect(v) {
				varr := em.fb.newIndirectRegister()
				em.fb.bindVarReg(v.Name, varr)
				addresses[i] = em.addressNewIndirectVar(varr, varType, pos, node.Type)
				continue
			}
			// Declare a local variable.
			varr := em.fb.newRegister(varType.Kind())
			em.fb.bindVarReg(v.Name, varr)
			addresses[i] = em.addressLocalVar(varr, varType, pos, node.Type)
		}
		em.assignValuesToAddresses(addresses, node.Rhs)
		return
	}

	// Emit an assignment.
	addresses := make([]address, len(node.Lhs))
	for i, v := range node.Lhs {
		pos := v.Pos()
		switch v := v.(type) {
		case *ast.Identifier:
			// Blank identifier.
			if isBlankIdentifier(v) {
				addresses[i] = em.addressBlankIdent(pos)
				break
			}
			varType := em.typ(v)
			// Package/closure/imported variable.
			if index, ok := em.varStore.nonLocalVarIndex(v); ok {
				addresses[i] = em.addressNonLocalVar(int16(index), varType, pos, node.Type)
				break
			}
			// Local variable.
			reg := em.fb.scopeLookup(v.Name)
			addresses[i] = em.addressLocalVar(reg, varType, pos, node.Type)
		case *ast.Index:
			exprType := em.typ(v.Expr)
			expr := em.emitExpr(v.Expr, exprType)
			indexType := intType
			if exprType.Kind() == reflect.Map {
				indexType = exprType.Key()
			}
			index := em.emitExpr(v.Index, indexType)
			if exprType.Kind() == reflect.Map {
				addresses[i] = em.addressMapIndex(expr, index, exprType, pos, node.Type)
			} else {
				addresses[i] = em.addressSliceIndex(expr, index, exprType, pos, node.Type)
			}
		case *ast.Selector:
			if index, ok := em.varStore.nonLocalVarIndex(v); ok {
				addresses[i] = em.addressNonLocalVar(int16(index), em.typ(v), pos, node.Type)
				break
			}
			typ := em.typ(v.Expr)
			reg := em.emitExpr(v.Expr, typ)
			field, _ := typ.FieldByName(v.Ident)
			index := em.fb.makeIntConstant(encodeFieldIndex(field.Index))
			addresses[i] = em.addressStructSelector(reg, index, typ, pos, node.Type)
			break
		case *ast.UnaryOperator:
			if v.Operator() != ast.OperatorPointer {
				panic("BUG.") // remove.
			}
			typ := em.typ(v.Expr)
			reg := em.emitExpr(v.Expr, typ)
			addresses[i] = em.addressPtrIndirect(reg, typ, pos, node.Type)
		default:
			panic("BUG.") // remove.
		}
	}
	em.assignValuesToAddresses(addresses, node.Rhs)
}

// emitImport emits an import node, returning the list of all 'init' functions
// emitted.
//
// TODO: the argument isTemplate must be passed explicitly because it's
// different from em.isTemplate. Why?
//
// TODO: this function works correctly but its code looks very ugly and hard
// to understand. Review and improve the code.
//
func (em *emitter) emitImport(node *ast.Import, isTemplate bool) []*runtime.Function {

	// If the imported package is predefined the emitter does not have to do
	// anything: the predefined values have already been added to the type infos
	// of the tree, and the init functions have already been called when gc
	// imported the predefined package.
	if node.Tree == nil {
		return nil
	}

	backupPkg := em.pkg
	var backupPath string
	var backupBuilder *functionBuilder
	if isTemplate {
		backupPath = em.fb.getPath()
		em.fb.changePath(node.Tree.Path)
		backupBuilder = em.fb
	}

	// Emit the package and collect functions, variables and init functions.
	pkg := node.Tree.Nodes[0].(*ast.Package)
	funcs, vars, inits := em.emitPackage(pkg, false, node.Tree.Path)

	blankImport := false

	if !isTemplate {
		em.pkg = backupPkg
	}
	var importName string
	if node.Ident == nil {
		importName = pkg.Name
		if isTemplate {
			// Imports without identifiers are handled as 'import . "path"'.
			importName = ""
		} else {
			importName = pkg.Name
		}
	} else {
		if isTemplate {
			importName = node.Ident.Name
			if node.Ident.Name == "." {
				importName = ""
			}
		}
		switch node.Ident.Name {
		case "_":
			blankImport = true
		case ".":
			importName = ""
		default:
			importName = node.Ident.Name
		}
	}

	var targetPkg *ast.Package
	if isTemplate {
		targetPkg = backupPkg
	} else {
		targetPkg = em.pkg
	}

	if !blankImport {
		// Make available the imported functions.
		for name, fn := range funcs {
			if importName != "" {
				name = importName + "." + name
			}
			em.fnStore.makeAvailableScriggoFn(targetPkg, name, fn)
		}

		// Add the imported variables.
		for name, v := range vars {
			if importName != "" {
				name = importName + "." + name
			}
			em.varStore.bindScriggoPackageVar(targetPkg, name, v)
		}
	}

	if isTemplate {
		em.fb = backupBuilder
		em.pkg = backupPkg
		em.fb.changePath(backupPath)
	}

	return inits
}

// emitSelect emits the 'select' statements. The emission is composed by 4 main
// parts:
//
// 1) Preparation of the channel and value registers for every case.
//
// 2) Emission of the 'case' instructions. Every case must be followed by a
// 'goto' which points to the respective case body below.
//
// 3) Emission of the 'select' instruction.
//
// 4) Emission of the assignment node, in case of a case with assignment, and of
// the body for every case.
//
func (em *emitter) emitSelect(selectNode *ast.Select) {

	// Emit an empty select.
	if len(selectNode.Cases) == 0 {
		em.fb.emitSelect()
		return
	}

	// Enter in a new stack: all the registers allocated during the execution of
	// the 'select' statement will be released at the end of it.
	em.fb.enterStack()

	chs := make([]int8, len(selectNode.Cases))
	ok := em.fb.newRegister(reflect.Bool)
	value := [4]int8{
		intRegister:     em.fb.newRegister(reflect.Int),
		floatRegister:   em.fb.newRegister(reflect.Float64),
		stringRegister:  em.fb.newRegister(reflect.String),
		generalRegister: em.fb.newRegister(reflect.Interface),
	}

	// Prepare the registers for the 'select' instruction.
	for i, cas := range selectNode.Cases {
		switch cas := cas.Comm.(type) {
		case nil: // default: nothing to do.
		case *ast.UnaryOperator:
			// <- ch
			chExpr := cas.Expr
			chs[i] = em.emitExpr(chExpr, em.typ(chExpr))
		case *ast.Assignment:
			// v [, ok ] = <- ch
			chExpr := cas.Rhs[0].(*ast.UnaryOperator).Expr
			chs[i] = em.emitExpr(chExpr, em.typ(chExpr))
		case *ast.Send:
			// ch <- v
			chExpr := cas.Channel
			chType := em.typ(chExpr)
			elemType := chType.Elem()
			chs[i] = em.emitExpr(chExpr, chType)
			em.emitExprR(cas.Value, elemType, value[kindToType(elemType.Kind())])
		}
	}

	// Emit all the 'case' instructions.
	casesLabel := make([]label, len(selectNode.Cases))
	for i, cas := range selectNode.Cases {
		casesLabel[i] = em.fb.newLabel()
		switch comm := cas.Comm.(type) {
		case nil:
			// default
			em.fb.emitCase(false, reflect.SelectDefault, 0, 0)
		case *ast.UnaryOperator:
			// <- ch
			em.fb.emitCase(false, reflect.SelectRecv, 0, chs[i])
		case *ast.Assignment:
			// v [, ok ] = <- ch
			chExpr := comm.Rhs[0].(*ast.UnaryOperator).Expr
			chType := em.typ(chExpr)
			elemType := chType.Elem()
			em.fb.emitCase(false, reflect.SelectRecv, value[kindToType(elemType.Kind())], chs[i])
		case *ast.Send:
			// ch <- v
			chExpr := comm.Channel
			chType := em.typ(chExpr)
			elemType := chType.Elem()
			em.fb.emitCase(false, reflect.SelectSend, value[kindToType(elemType.Kind())], chs[i])
		}
		em.fb.emitGoto(casesLabel[i])
	}

	// Emit the 'select' instruction.
	em.fb.emitSelect()

	// Emit bodies of the 'select' cases.
	casesEnd := em.fb.newLabel()
	for i, cas := range selectNode.Cases {
		// Make the previous 'goto' point here.
		em.fb.setLabelAddr(casesLabel[i])
		// Emit an assignment if it is a receive case with an assignment.
		if assignment, isAssignment := cas.Comm.(*ast.Assignment); isAssignment {
			receiveExpr := assignment.Rhs[0].(*ast.UnaryOperator)
			chExpr := receiveExpr.Expr
			elemType := em.typ(chExpr).Elem()
			// Split the assignment in the received value and the ok value if this exists.
			em.fb.bindVarReg("$chanElem", value[kindToType(elemType.Kind())])
			pos := chExpr.Pos()
			valueExpr := ast.NewIdentifier(pos, "$chanElem")
			em.typeInfos[valueExpr] = em.typeInfos[receiveExpr]
			valueAssignment := ast.NewAssignment(pos, assignment.Lhs[0:1], assignment.Type, []ast.Expression{valueExpr})
			em.emitAssignmentNode(valueAssignment)
			if len(assignment.Lhs) == 2 { // case has 'ok'
				em.fb.emitMove(true, 1, ok, reflect.Bool, false)
				em.fb.emitIf(false, 0, runtime.ConditionOK, 0, reflect.Interface, assignment.Pos())
				em.fb.emitMove(true, 0, ok, reflect.Bool, false)
				okExpr := ast.NewIdentifier(pos, "$ok")
				em.typeInfos[okExpr] = &typeInfo{
					Type: boolType,
				}
				em.fb.bindVarReg("$ok", ok)
				okAssignment := ast.NewAssignment(pos, assignment.Lhs[1:2], assignment.Type, []ast.Expression{okExpr})
				em.emitAssignmentNode(okAssignment)
			}
		}
		// Emit the nodes of the body of the case.
		em.emitNodes(cas.Body)
		// All 'case' bodies jump to the end of the 'select' bodies, except for the last one.
		if i < len(selectNode.Cases)-1 {
			em.fb.emitGoto(casesEnd)
		}
	}
	em.fb.setLabelAddr(casesEnd)

	// Release all the registers allocated during the execution of the 'select'
	// statement.
	em.fb.exitStack()

}

// emitSwitch emits instructions for a switch node.
func (em *emitter) emitSwitch(node *ast.Switch) {

	em.fb.enterScope()

	if node.Init != nil {
		em.emitNodes([]ast.Node{node.Init})
	}

	var expr int8
	var typ reflect.Type

	if node.Expr == nil {
		typ = boolType
		expr = em.fb.newRegister(typ.Kind())
		em.fb.emitMove(true, 1, expr, typ.Kind(), true)
		node.Expr = ast.NewIdentifier(node.Pos(), "true")
		em.typeInfos[node.Expr] = &typeInfo{
			Constant:   boolConst(true),
			Type:       boolType,
			value:      int64(1), // true
			valueType:  boolType,
			Properties: propertyUntyped | propertyHasValue,
		}
	} else {
		typ = em.typ(node.Expr)
		expr = em.emitExpr(node.Expr, typ)
	}

	bodyLabels := make([]label, len(node.Cases))
	endSwitchLabel := em.fb.newLabel()

	var defaultLabel label
	hasDefault := false

	for i, cas := range node.Cases {
		bodyLabels[i] = em.fb.newLabel()
		hasDefault = hasDefault || cas.Expressions == nil
		for _, caseExpr := range cas.Expressions {
			em.fb.enterStack()
			pos := caseExpr.Pos()
			binOp := ast.NewBinaryOperator(pos, ast.OperatorNotEqual, node.Expr, caseExpr)
			em.typeInfos[binOp] = &typeInfo{
				Type: boolType,
			}
			em.emitCondition(binOp)
			em.fb.exitStack()
			em.fb.emitGoto(bodyLabels[i])
		}
	}

	if hasDefault {
		defaultLabel = em.fb.newLabel()
		em.fb.emitGoto(defaultLabel)
	} else {
		em.fb.emitGoto(endSwitchLabel)
	}

	for i, cas := range node.Cases {
		if cas.Expressions == nil {
			em.fb.setLabelAddr(defaultLabel)
		}
		em.fb.setLabelAddr(bodyLabels[i])
		em.fb.enterScope()
		em.emitNodes(cas.Body)
		hasFallthrough := false
		for i := len(cas.Body) - 1; i >= 0; i-- {
			if _, ok := cas.Body[i].(*ast.Fallthrough); ok {
				hasFallthrough = true
				break
			}
		}
		if !hasFallthrough {
			em.fb.emitGoto(endSwitchLabel)
		}
		em.fb.exitScope()
	}

	em.fb.setLabelAddr(endSwitchLabel)

	em.fb.exitScope()

	return
}

// emitTypeSwitch emits instructions for a type switch node.
func (em *emitter) emitTypeSwitch(node *ast.TypeSwitch) {

	em.fb.enterScope()

	// Emit the init simple statement, if present.
	if node.Init != nil {
		em.emitNodes([]ast.Node{node.Init})
	}

	// Emit the type switch guard expression.
	guardExpr := node.Assignment.Rhs[0].(*ast.TypeAssertion).Expr
	expr := em.emitExpr(guardExpr, em.typ(guardExpr))

	// Store the name of the variable declared in the guard, if present.
	guardNewVar := ""
	if len(node.Assignment.Lhs) == 1 {
		guardNewVar = node.Assignment.Lhs[0].(*ast.Identifier).Name
	}

	var intReg int8
	var floatReg int8
	var stringReg int8
	var generalReg int8

	// Allocate only the necessary register.
	// Note that 'expr' has already been allocated; these registers are
	// necessary only when the type switch declares a new variable and such
	// variable is used in a case body with just one type.
	if guardNewVar != "" {
		for _, clause := range node.Cases {
			if types := clause.Expressions; len(types) == 1 && !em.isPredeclNil(types[0]) {
				switch kindToType(em.ti(clause.Expressions[0]).Type.Kind()) {
				case intRegister:
					if intReg == 0 {
						intReg = em.fb.newRegister(reflect.Int)
					}
				case floatRegister:
					if floatReg == 0 {
						floatReg = em.fb.newRegister(reflect.Float64)
					}
				case stringRegister:
					if stringReg == 0 {
						stringReg = em.fb.newRegister(reflect.String)
					}
				case generalRegister:
					if generalReg == 0 {
						generalReg = em.fb.newRegister(reflect.Interface)
					}
				}
			}
		}
	}

	clauseBody := make([]label, len(node.Cases))
	end := em.fb.newLabel()

	hasDefault := false

	for i, clause := range node.Cases {
		clauseBody[i] = em.fb.newLabel()
		switch {
		case isDefault(clause):
			hasDefault = true
		case len(clause.Expressions) == 1:
			if em.isPredeclNil(clause.Expressions[0]) {
				em.fb.emitIf(false, expr, runtime.ConditionInterfaceNil, 0, reflect.Interface, clause.Expressions[0].Pos())
			} else {
				typ := em.ti(clause.Expressions[0]).Type
				var reg int8
				switch kindToType(typ.Kind()) {
				case intRegister:
					reg = intReg
				case floatRegister:
					reg = floatReg
				case stringRegister:
					reg = stringReg
				case generalRegister:
					reg = generalReg
				}
				em.fb.emitAssert(expr, typ, reg)
			}
			next := em.fb.newLabel()
			em.fb.emitGoto(next)          // assert failed
			em.fb.emitGoto(clauseBody[i]) // assert ok
			em.fb.setLabelAddr(next)
		default: // case type1, type2 .. typeN:
			for _, typExpr := range clause.Expressions {
				if em.isPredeclNil(typExpr) {
					em.fb.emitIf(false, expr, runtime.ConditionInterfaceNil, 0, reflect.Interface, typExpr.Pos())
				} else {
					typ := em.ti(typExpr).Type
					em.fb.emitAssert(expr, typ, 0)
				}
				nextClause := em.fb.newLabel()
				em.fb.emitGoto(nextClause)    // assert failed
				em.fb.emitGoto(clauseBody[i]) // assert ok
				em.fb.setLabelAddr(nextClause)
			}
		}
	}

	// Jump to the default case (if present) or to the end.
	var defaultClause label
	if hasDefault {
		defaultClause = em.fb.newLabel()
		em.fb.emitGoto(defaultClause)
	} else {
		em.fb.emitGoto(end)
	}

	// Emit the bodies of the case clauses.
	for i, clause := range node.Cases {
		if isDefault(clause) {
			em.fb.setLabelAddr(defaultClause)
		}
		em.fb.setLabelAddr(clauseBody[i])
		em.fb.enterScope()
		if guardNewVar != "" {
			if len(clause.Expressions) == 1 && !em.isPredeclNil(clause.Expressions[0]) {
				switch kindToType(em.ti(clause.Expressions[0]).Type.Kind()) {
				case intRegister:
					em.fb.bindVarReg(guardNewVar, intReg)
				case floatRegister:
					em.fb.bindVarReg(guardNewVar, floatReg)
				case stringRegister:
					em.fb.bindVarReg(guardNewVar, stringReg)
				case generalRegister:
					em.fb.bindVarReg(guardNewVar, generalReg)
				}
			} else {
				em.fb.bindVarReg(guardNewVar, expr)
			}
		}
		em.emitNodes(clause.Body)
		em.fb.exitScope()
		em.fb.emitGoto(end)
	}

	em.fb.setLabelAddr(end)
	em.fb.exitScope()

}
