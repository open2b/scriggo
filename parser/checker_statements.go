package parser

import "scrigo/ast"

func (tc *typechecker) checkNodes(nodes []ast.Node) {

	for _, node := range nodes {

		switch node := node.(type) {

		case *ast.Block:

			tc.NewScope()
			tc.checkNodes(node.Nodes)
			tc.RemoveCurrentScope()

		case *ast.If:

			tc.NewScope()
			tc.checkAssignment(node.Assignment)
			expr := tc.checkExpression(node.Condition)
			// TODO (Gianluca): to review.
			_ = expr
			// if condTyp.Type != boolType {
			// 	return tc.errorf(node.Condition, "non-bool %s (type %s) used as if condition", node.Condition, condTyp)
			// }
			if node.Then != nil {
				tc.NewScope()
				tc.checkNodes(node.Then.Nodes)
				tc.RemoveCurrentScope()
			}
			if node.Else != nil {
				switch els := node.Else.(type) {
				case *ast.Block:
					tc.NewScope()
					tc.checkNodes(els.Nodes)
					tc.RemoveCurrentScope()
				case *ast.If:
					// TODO (Gianluca): same problem we had in renderer:
					tc.checkNodes([]ast.Node{els})
				}
			}
			tc.RemoveCurrentScope()

		case *ast.For:

			tc.NewScope()
			tc.checkAssignment(node.Init)
			expr := tc.checkExpression(node.Condition)
			// TODO (Gianluca): to review.
			_ = expr
			// if expr.Type != boolType {
			// 	return tc.errorf(node.Condition, "non-bool %s (type %s) used as for condition", node.Condition, condTyp)
			// }
			tc.checkAssignment(node.Post)
			tc.NewScope()
			tc.checkNodes(node.Body)
			tc.RemoveCurrentScope()
			tc.RemoveCurrentScope()

		case *ast.ForRange:

			tc.NewScope()
			tc.checkAssignment(node.Assignment)
			tc.NewScope()
			tc.checkNodes(node.Body)
			tc.RemoveCurrentScope()
			tc.RemoveCurrentScope()

		case *ast.Assignment:

			switch node.Type {

			case ast.AssignmentIncrement, ast.AssignmentDecrement:

				v := node.Variables[0]
				exprType := tc.checkExpression(v)
				// TODO (Gianluca): to review.
				_ = exprType
				// if !exprType.Numeric() {
				// 	panic(tc.errorf(node, "invalid operation: %v (non-numeric type %s)", v, exprType))
				// }
				return

			case ast.AssignmentAddition, ast.AssignmentSubtraction, ast.AssignmentMultiplication,
				ast.AssignmentDivision, ast.AssignmentModulo:

				lv := node.Variables[0]
				lt := tc.checkExpression(lv)
				// TODO (Gianluca): to review.
				_ = lt
				// if !lt.Numeric() {
				// 	panic(tc.errorf(node, "invalid operation: %v (non-numeric type %s)", lv, lt))
				// }

				rv := node.Values[0]
				err := tc.assignValueToVariable(node, lv, rv, nil, false, false)
				if err != nil {
					panic(err)
				}

			case ast.AssignmentSimple, ast.AssignmentDeclaration:

				tc.checkAssignment(node)

			}

		case *ast.Identifier:

			t := tc.evalIdentifier(node)
			if t.IsPackage() {
				panic(tc.errorf(node, "use of package %s without selector", t))
			}
			panic(tc.errorf(node, "%s evaluated but not used", node.Name))

		case *ast.Break, *ast.Continue:

		case *ast.Return:

			// TODO (Gianluca): should check if return is expected?
			// TODO (Gianluca): check if return's value is the same of the function where return is in.
			panic("not implemented")

		case *ast.Switch:

			tc.NewScope()
			tc.checkAssignment(node.Init)
			for _, cas := range node.Cases {
				err := tc.checkCase(cas, false, node.Expr)
				if err != nil {
					panic(err)
				}
			}
			tc.RemoveCurrentScope()

		case *ast.TypeSwitch:

			tc.NewScope()
			tc.checkAssignment(node.Init)
			tc.checkAssignment(node.Assignment)
			for _, cas := range node.Cases {
				err := tc.checkCase(cas, true, nil)
				if err != nil {
					panic(err)
				}
			}
			tc.RemoveCurrentScope()

		case *ast.Const, *ast.Var:

			tc.checkAssignment(node)

		default:

			panic("not implemented") // TODO (Gianluca): review!

		}

	}

}

func (tc *typechecker) checkCase(node *ast.Case, isTypeSwitch bool, switchExpr ast.Expression) error {
	tc.NewScope()
	switchExprTyp := tc.typeof(switchExpr, noEllipses)
	for _, expr := range node.Expressions {
		cas := tc.typeof(expr, noEllipses)
		if isTypeSwitch && !switchExprTyp.IsType() {
			return tc.errorf(expr, "%v (type %v) is not a type", expr, cas.Type)
		}
		if !isTypeSwitch && switchExprTyp.IsType() {
			return tc.errorf(expr, "type %v is not an expression", cas.Type)
		}
		if cas.Type != switchExprTyp.Type {
			return tc.errorf(expr, "invalid case %v in switch on %v (mismatched types %v and %v)", expr, switchExpr, cas.Type, switchExprTyp.Type)
		}
	}
	tc.checkNodes(node.Body)
	tc.RemoveCurrentScope()
	return nil
}

// TODO (Gianluca): handle "isConst"
// TODO (Gianluca): typ doesn't get the type zero, just checks if type is
// correct when a value is provided. Implement "var a int"
func (tc *typechecker) assignValueToVariable(node ast.Node, variable, value ast.Expression, typ *ast.TypeInfo, isDeclaration, isConst bool) error {
	variableType := tc.checkExpression(variable)
	valueType := tc.checkExpression(value)
	if isConst && (valueType.Constant == nil) {
		return tc.errorf(node, "const initializer is not a constant") // TODO (Gianluca): to review.
	}
	if typ != valueType {
		return tc.errorf(node, "canont use %v (type %v) as type %v in assignment", value, valueType, typ)
	}
	// TODO (Gianluca): to review.
	_ = variableType
	// err = valueType.MustAssignableTo(variableType.ReflectType())
	// if err != nil {
	// 	return err
	// }

	if isDeclaration {
		if ident, ok := variable.(*ast.Identifier); ok {
			_, alreadyDefined := tc.LookupScope(ident.Name, true) // TODO (Gianluca): to review.
			if alreadyDefined {
				return tc.errorf(node, "no new variables on left side of :=")
			}
			tc.AssignScope(ident.Name, valueType)
			return nil
		}
		panic("bug/not implemented") // TODO (Gianluca): can we have a declaration without an identifier?
	}
	return nil
}

// TODO (Gianluca): manage builtin functions.
func (tc *typechecker) checkAssignmentWithCall(node ast.Node, variables []ast.Expression, call *ast.Call, typ *ast.TypeInfo, isDeclaration, isConst bool) error {
	values := tc.checkCallExpression(call, false) // TODO (Gianluca): is "false" correct?
	if len(variables) != len(values) {
		return tc.errorf(node, "assignment mismatch: %d variables but %v returns %d values", len(variables), call, len(values))
	}
	for i := range variables {
		// TODO (Gianluca): il secondo 'variables[i]' dovrebbe in realtà essere
		// 'values[i]', ma attualmente non compila.
		err := tc.assignValueToVariable(node, variables[i], variables[i], typ, isDeclaration, isConst)
		if err != nil {
			return err
		}
	}
	return nil
}

// TODO (Gianluca): handle
//		 var a, b int = f() // func f() (int, string)
// (should be automatically handled, to verify)
func (tc *typechecker) checkAssignment(node ast.Node) {

	var variables, values []ast.Expression
	var typ *ast.TypeInfo
	var isDeclaration, isConst bool

	switch n := node.(type) {
	case *ast.Var:
		variables = make([]ast.Expression, len(n.Identifiers))
		for i, ident := range n.Identifiers {
			variables[i] = ident
		}
		values = n.Values
		isDeclaration = true
	case *ast.Const:
		variables = make([]ast.Expression, len(n.Identifiers))
		for i, ident := range n.Identifiers {
			variables[i] = ident
		}
		values = n.Values
		isConst = true
		isDeclaration = true
	case *ast.Assignment:
		variables = n.Variables
		values = n.Values
		isDeclaration = n.Type == ast.AssignmentDeclaration
	}

	if len(variables) == 1 && len(values) == 1 {
		err := tc.assignValueToVariable(node, variables[0], values[0], typ, isDeclaration, isConst)
		if err != nil {
			panic(err)
		}
		return
	}
	if len(values) == 0 && typ != nil {
		for i := range variables {
			// TODO (Gianluca): zero must contain the zero of type "typ".
			zero := (ast.Expression)(nil)
			if isConst || isDeclaration {
				panic("bug?") // TODO (Gianluca): review!
			}
			err := tc.assignValueToVariable(node, variables[i], zero, typ, false, false)
			if err != nil {
				panic(err)
			}
		}
	}
	if len(variables) == len(values) {
		for i := range variables {
			// TODO (Gianluca): if all variables have already been declared
			// previously, a declaration must end with an error.
			// _, alreadyDefined := tc.LookupScope("variable name", true) // TODO (Gianluca): to review.
			alreadyDefined := false
			isDecl := isDeclaration && !alreadyDefined
			err := tc.assignValueToVariable(node, variables[i], values[i], typ, isDecl, isConst)
			if err != nil {
				panic(err)
			}
		}
		return
	}
	if len(variables) == 2 && len(values) == 1 {
		switch value := values[0].(type) {
		case *ast.Call:
			err := tc.checkAssignmentWithCall(node, variables, value, typ, isDeclaration, isConst)
			if err != nil {
				panic(err)
			}
		case *ast.TypeAssertion:
			// TODO (Gianluca):
			// Se la type assertion è valida,
			// if type assertion is valid:
			// 		first value is type assertion type
			//		second value is always a boolean
		case *ast.Index:
			// TODO (Gianluca):
			// if indexing a map && key is valid ... :
			//		first value is maptype.Elem()
			// 		second value is always a boolean
		}
		return
	}
	if len(variables) > 2 && len(values) == 1 {
		call, ok := values[0].(*ast.Call)
		if ok {
			err := tc.checkAssignmentWithCall(node, variables, call, typ, isDeclaration, isConst)
			if err != nil {
				panic(err)
			}
		}
	}
	panic(tc.errorf(node, "assignment mismatch: %d variable but %d values", len(variables), len(values)))
}
