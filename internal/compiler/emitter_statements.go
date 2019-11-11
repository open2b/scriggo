// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"reflect"
	"scriggo/ast"
	"scriggo/runtime"
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
			em.fb.emitContinue(em.rangeLabels[len(em.rangeLabels)-1][0])

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
				if node.Ident != nil && node.Ident.Name == "_" {
					// Nothing to do: template pages cannot have
					// collateral effects.
				} else {
					backupPath := em.fb.getPath()
					em.fb.changePath(node.Tree.Path)
					backupBuilder := em.fb
					backupPkg := em.pkg
					functions, vars, inits := em.emitPackage(node.Tree.Nodes[0].(*ast.Package), false, node.Path)
					var importName string
					if node.Ident == nil {
						// Imports without identifiers are handled as 'import . "path"'.
						importName = ""
					} else {
						importName = node.Ident.Name
						if node.Ident.Name == "." {
							importName = ""
						}
					}
					if em.functions[backupPkg] == nil {
						em.functions[backupPkg] = map[string]*runtime.Function{}
					}
					for name, fn := range functions {
						if importName == "" {
							em.functions[backupPkg][name] = fn
						} else {
							em.functions[backupPkg][importName+"."+name] = fn
						}
					}
					if em.availableVarIndexes[backupPkg] == nil {
						em.availableVarIndexes[backupPkg] = map[string]int16{}
					}
					for name, v := range vars {
						if importName == "" {
							em.availableVarIndexes[backupPkg][name] = v
						} else {
							em.availableVarIndexes[backupPkg][importName+"."+name] = v
						}
					}
					if len(inits) > 0 {
						panic("BUG: have inits!") // remove.
					}
					em.fb = backupBuilder
					em.pkg = backupPkg
					em.fb.changePath(backupPath)
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
				forLabel := em.fb.newLabel()
				em.fb.setLabelAddr(forLabel)
				em.emitCondition(node.Condition)
				endForLabel := em.fb.newLabel()
				em.fb.emitGoto(endForLabel)
				em.emitNodes(node.Body)
				if node.Post != nil {
					em.emitNodes([]ast.Node{node.Post})
				}
				em.fb.emitGoto(forLabel)
				em.fb.setLabelAddr(endForLabel)
			} else {
				forLabel := em.fb.newLabel()
				em.fb.setLabelAddr(forLabel)
				em.emitNodes(node.Body)
				if node.Post != nil {
					em.emitNodes([]ast.Node{node.Post})
				}
				em.fb.emitGoto(forLabel)
			}
			em.fb.exitScope()
			if em.breakLabel != nil {
				em.fb.setLabelAddr(*em.breakLabel)
			}
			em.breakable = currentBreakable
			em.breakLabel = currentBreakLabel

		case *ast.ForRange:
			em.fb.enterScope()
			vars := node.Assignment.Lhs
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
				typ := em.ti(vars[1]).Type
				name := vars[1].(*ast.Identifier).Name
				if node.Assignment.Type == ast.AssignmentDeclaration {
					elem = em.fb.newRegister(typ.Kind())
					em.fb.bindVarReg(name, elem)
				} else {
					elem = em.fb.scopeLookup(name)
				}
			}
			expr := node.Assignment.Rhs[0]
			exprType := em.ti(expr).Type
			exprReg, kExpr := em.emitExprK(expr, exprType)
			if exprType.Kind() != reflect.String && kExpr {
				kExpr = false
				exprReg = em.emitExpr(expr, exprType)
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
			if node.Assignment != nil {
				em.emitNodes([]ast.Node{node.Assignment})
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

		case *ast.Include:
			path := em.fb.getPath()
			em.fb.changePath(node.Tree.Path)
			em.emitNodes(node.Tree.Nodes)
			em.fb.changePath(path)

		case *ast.Label:
			if _, found := em.labels[em.fb.fn][node.Name.Name]; !found {
				if em.labels[em.fb.fn] == nil {
					em.labels[em.fb.fn] = make(map[string]label)
				}
				em.labels[em.fb.fn][node.Name.Name] = em.fb.newLabel()
			}
			em.fb.setLabelAddr(em.labels[em.fb.fn][node.Name.Name])
			if node.Statement != nil {
				em.emitNodes([]ast.Node{node.Statement})
			}

		case *ast.Return:
			offset := [4]int8{}
			for i, v := range node.Values {
				typ := em.fb.fn.Type.Out(i)
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
			chanType := em.ti(node.Channel).Type
			chann := em.emitExpr(node.Channel, chanType)
			value := em.emitExpr(node.Value, em.ti(node.Value).Type)
			em.fb.emitSend(chann, value, node.Pos(), chanType.Elem().Kind())

		case *ast.Show:
			// render([implicit *vm.Env,] gD io.Writer, gE interface{}, iA ast.Context)
			em.emitExprR(node.Expr, emptyInterfaceType, em.fb.templateRegs.gE)
			em.fb.emitMove(true, int8(node.Context), em.fb.templateRegs.iA, reflect.Int)
			if em.inURL {
				// In a URL context: use the urlWriter, that implements io.Writer.
				em.fb.emitMove(false, em.fb.templateRegs.gF, em.fb.templateRegs.gD, reflect.Interface)
			} else {
				// Not in a URL context: use the default writer.
				em.fb.emitMove(false, em.fb.templateRegs.gA, em.fb.templateRegs.gD, reflect.Interface)
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
					em.fb.emitMethodValue("WriteText", em.fb.templateRegs.gF, writeFun)
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
			em.fb.emitMethodValue("StartURL", em.fb.templateRegs.gF, method)
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
			for i, v := range node.Lhs {
				if isBlankIdentifier(v) {
					addresses[i] = em.newAddress(addressBlank, reflect.Type(nil), 0, 0, v.Pos())
				} else {
					staticType := em.ti(v).Type
					if em.indirectVars[v] {
						varr := -em.fb.newRegister(reflect.Interface)
						em.fb.bindVarReg(v.Name, varr)
						addresses[i] = em.newAddress(addressIndirectDeclaration, staticType, varr, 0, v.Pos())
					} else {
						varr := em.fb.newRegister(staticType.Kind())
						em.fb.bindVarReg(v.Name, varr)
						addresses[i] = em.newAddress(addressLocalVariable, staticType, varr, 0, v.Pos())
					}
				}
			}
			em.assignValuesToAddresses(addresses, node.Rhs)

		case ast.Expression:
			em.emitExprR(node, reflect.Type(nil), 0)

		default:
			panic(fmt.Sprintf("BUG: node %T not supported", node)) // remove.

		}

	}

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

	// Create some shared registers; preallocation is not a problem: when the
	// select statement will be ended, all registers will be released.
	ch := em.fb.newRegister(reflect.Chan)
	ok := em.fb.newRegister(reflect.Bool)
	value := [4]int8{
		intRegister:     em.fb.newRegister(reflect.Int),
		floatRegister:   em.fb.newRegister(reflect.Float64),
		stringRegister:  em.fb.newRegister(reflect.String),
		generalRegister: em.fb.newRegister(reflect.Interface),
	}

	// Prepare the registers for the 'select' instruction.
	for _, cas := range selectNode.Cases {
		switch cas := cas.Comm.(type) {
		case nil: // default: nothing to do.
		case *ast.UnaryOperator:
			// <- ch
			chExpr := cas.Expr
			em.emitExprR(chExpr, em.ti(chExpr).Type, ch)
		case *ast.Assignment:
			// v [, ok ] = <- ch
			chExpr := cas.Rhs[0].(*ast.UnaryOperator).Expr
			em.emitExprR(chExpr, em.ti(chExpr).Type, ch)
		case *ast.Send:
			// ch <- v
			chExpr := cas.Channel
			chType := em.ti(chExpr).Type
			elemType := chType.Elem()
			em.emitExprR(chExpr, chType, ch)
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
			em.fb.emitCase(false, reflect.SelectDefault, 0, 0, reflect.Invalid)
		case *ast.UnaryOperator:
			// <- ch
			chExpr := comm.Expr
			elemType := em.ti(chExpr).Type.Elem()
			em.fb.emitCase(false, reflect.SelectRecv, 0, ch, elemType.Kind())
		case *ast.Assignment:
			// v [, ok ] = <- ch
			chExpr := comm.Rhs[0].(*ast.UnaryOperator).Expr
			chType := em.ti(chExpr).Type
			elemType := chType.Elem()
			em.fb.emitCase(false, reflect.SelectRecv, value[kindToType(elemType.Kind())], ch, elemType.Kind())
		case *ast.Send:
			// ch <- v
			chExpr := comm.Channel
			chType := em.ti(chExpr).Type
			elemType := chType.Elem()
			em.fb.emitCase(false, reflect.SelectSend, value[kindToType(elemType.Kind())], ch, elemType.Kind())
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
			chType := em.ti(chExpr).Type
			elemType := chType.Elem()
			// Split the assignment in the received value and the ok value if this exists.
			em.fb.bindVarReg("$chanElem", value[kindToType(elemType.Kind())])
			pos := chExpr.Pos()
			valueExpr := ast.NewIdentifier(pos, "$chanElem")
			em.typeInfos[valueExpr] = em.typeInfos[receiveExpr]
			valueAssignment := ast.NewAssignment(pos, assignment.Lhs[0:1], assignment.Type, []ast.Expression{valueExpr})
			em.emitAssignmentNode(valueAssignment)
			if len(assignment.Lhs) == 2 { // case has 'ok'
				em.fb.emitMove(true, 1, ok, reflect.Bool)
				em.fb.emitIf(false, 0, runtime.ConditionOK, 0, reflect.Interface, assignment.Pos())
				em.fb.emitMove(true, 0, ok, reflect.Bool)
				okExpr := ast.NewIdentifier(pos, "$ok")
				em.typeInfos[okExpr] = &TypeInfo{
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
		em.fb.emitMove(true, 1, expr, typ.Kind())
		node.Expr = ast.NewIdentifier(node.Pos(), "true")
		em.typeInfos[node.Expr] = &TypeInfo{
			Constant:   boolConst(true),
			Type:       boolType,
			value:      int64(1), // true
			valueType:  boolType,
			Properties: PropertyUntyped | PropertyHasValue,
		}
	} else {
		typ = em.ti(node.Expr).Type
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
			pos := caseExpr.Pos()
			binOp := ast.NewBinaryOperator(pos, ast.OperatorNotEqual, node.Expr, caseExpr)
			em.typeInfos[binOp] = &TypeInfo{
				Type: boolType,
			}
			em.emitCondition(binOp)
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

	if node.Init != nil {
		em.emitNodes([]ast.Node{node.Init})
	}

	typeAssertion := node.Assignment.Rhs[0].(*ast.TypeAssertion)
	expr := em.emitExpr(typeAssertion.Expr, em.ti(typeAssertion.Expr).Type)

	bodyLabels := make([]label, len(node.Cases))
	endSwitchLabel := em.fb.newLabel()

	var defaultLabel label
	hasDefault := false

	for i, cas := range node.Cases {
		bodyLabels[i] = em.fb.newLabel()
		hasDefault = hasDefault || cas.Expressions == nil
		if len(cas.Expressions) == 1 {
			// If the type switch has an assignment, assign to the variable
			// using the type of the case.
			pos := cas.Expressions[0].Pos()
			if len(node.Assignment.Lhs) == 1 && !em.ti(cas.Expressions[0]).Nil() {
				ta := ast.NewTypeAssertion(pos, typeAssertion.Expr, cas.Expressions[0])
				em.typeInfos[ta] = &TypeInfo{
					Type: em.ti(cas.Expressions[0]).Type,
				}
				blank := ast.NewIdentifier(pos, "_")
				n := ast.NewAssignment(pos,
					[]ast.Expression{
						node.Assignment.Lhs[0], // the variable
						blank,                  // a dummy blank identifier that prevent panicking
					},
					node.Assignment.Type,
					[]ast.Expression{ta},
				)
				em.emitNodes([]ast.Node{n})
			}
		} else {
			// If the type switch has an assignment, assign to the variable
			// keeping the type. Note that this assignment does not involve type
			// assertion statements, just takes the expression from the type
			// assertion of the type switch.
			if len(node.Assignment.Lhs) == 1 {
				pos := node.Assignment.Lhs[0].Pos()
				n := ast.NewAssignment(pos,
					[]ast.Expression{node.Assignment.Lhs[0]},
					node.Assignment.Type,
					[]ast.Expression{typeAssertion.Expr},
				)
				em.emitNodes([]ast.Node{n})
			}
		}
		for _, caseExpr := range cas.Expressions {
			if em.ti(caseExpr).Nil() {
				pos := caseExpr.Pos()
				em.fb.emitIf(false, expr, runtime.ConditionInterfaceNil, 0, reflect.Interface, pos)
			} else {
				caseType := em.ti(caseExpr).Type
				em.fb.emitAssert(expr, caseType, 0)
			}
			next := em.fb.newLabel()
			em.fb.emitGoto(next)
			em.fb.emitGoto(bodyLabels[i])
			em.fb.setLabelAddr(next)
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
		em.fb.exitScope()
		em.fb.emitGoto(endSwitchLabel)
	}

	em.fb.setLabelAddr(endSwitchLabel)
	em.fb.exitScope()

	return
}
