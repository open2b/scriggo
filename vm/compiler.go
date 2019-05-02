// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

import (
	"fmt"
	"reflect"

	"scrigo/ast"
	"scrigo/parser"
)

// A Compiler compiles sources generating VM's packages.
type Compiler struct {
	parser           *parser.Parser
	currentFunction  *ScrigoFunction
	typeinfo         map[ast.Node]*parser.TypeInfo
	fb               *FunctionBuilder // current function builder.
	importableGoPkgs map[string]*parser.GoPackage

	availableScrigoFunctions map[string]*ScrigoFunction
	availableNativeFunctions map[string]*NativeFunction
	availableVariables       map[string]variable

	// TODO (Gianluca): find better names.
	// TODO (Gianluca): do these maps have to have a *ScrigoFunction key or
	// can they be related to currentFunction in some way?
	assignedIndexesOfScrigoFunctions map[*ScrigoFunction]map[*ScrigoFunction]int8
	assignedIndexesOfNativeFunctions map[*ScrigoFunction]map[*NativeFunction]int8
	assignedIndexesOfVariables       map[*ScrigoFunction]map[variable]uint8

	isNativePkg map[string]bool
}

// NewCompiler returns a new compiler reading sources from r.
// Native (Go) packages are made available for importing.
func NewCompiler(r parser.Reader, packages map[string]*parser.GoPackage) *Compiler {
	c := &Compiler{
		importableGoPkgs: packages,

		availableScrigoFunctions: map[string]*ScrigoFunction{},
		availableNativeFunctions: map[string]*NativeFunction{},
		availableVariables:       map[string]variable{},

		assignedIndexesOfScrigoFunctions: map[*ScrigoFunction]map[*ScrigoFunction]int8{},
		assignedIndexesOfNativeFunctions: map[*ScrigoFunction]map[*NativeFunction]int8{},
		assignedIndexesOfVariables:       map[*ScrigoFunction]map[variable]uint8{},

		isNativePkg: map[string]bool{},
	}
	c.parser = parser.New(r, packages, true)
	return c
}

func (c *Compiler) Compile(path string) (*ScrigoFunction, error) {
	tree, err := c.parser.Parse(path, ast.ContextNone)
	if err != nil {
		return nil, err
	}
	tci := c.parser.TypeCheckInfos()
	c.typeinfo = tci[path].TypeInfo
	node := tree.Nodes[0].(*ast.Package)
	c.compilePackage(node)
	fun := c.currentFunction
	return fun, nil
}

// scrigoFunctionIndex returns fun's index inside current function, creating it
// if not exists.
func (c *Compiler) scrigoFunctionIndex(fun *ScrigoFunction) int8 {
	currFun := c.currentFunction
	i, ok := c.assignedIndexesOfScrigoFunctions[currFun][fun]
	if ok {
		return i
	}
	i = int8(len(currFun.scrigoFunctions))
	currFun.scrigoFunctions = append(currFun.scrigoFunctions, fun)
	if c.assignedIndexesOfScrigoFunctions[currFun] == nil {
		c.assignedIndexesOfScrigoFunctions[currFun] = make(map[*ScrigoFunction]int8)
	}
	c.assignedIndexesOfScrigoFunctions[currFun][fun] = i
	return i
}

// nativeFunctionIndex returns fun's index inside current function, creating it
// if not exists.
func (c *Compiler) nativeFunctionIndex(fun *NativeFunction) int8 {
	currFun := c.currentFunction
	i, ok := c.assignedIndexesOfNativeFunctions[currFun][fun]
	if ok {
		return i
	}
	i = int8(len(currFun.nativeFunctions))
	currFun.nativeFunctions = append(currFun.nativeFunctions, fun)
	if c.assignedIndexesOfNativeFunctions[currFun] == nil {
		c.assignedIndexesOfNativeFunctions[currFun] = make(map[*NativeFunction]int8)
	}
	c.assignedIndexesOfNativeFunctions[currFun][fun] = i
	return i
}

// variableIndex returns v's index inside current function, creating it if not
// exists.
func (c *Compiler) variableIndex(v variable) uint8 {
	currFun := c.currentFunction
	i, ok := c.assignedIndexesOfVariables[currFun][v]
	if ok {
		return i
	}
	i = uint8(len(currFun.variables))
	currFun.variables = append(currFun.variables, v)
	if c.assignedIndexesOfVariables[currFun] == nil {
		c.assignedIndexesOfVariables[currFun] = make(map[variable]uint8)
	}
	c.assignedIndexesOfVariables[currFun][v] = i
	return i
}

// compilePackage compiles pkg.
func (c *Compiler) compilePackage(pkg *ast.Package) {

	haveVariables := false
	for _, dec := range pkg.Declarations {
		_, ok := dec.(*ast.Var)
		if ok {
			haveVariables = true
			break
		}
	}

	var initVars *ScrigoFunction
	if haveVariables {
		initVars = NewScrigoFunction("main", "init.vars", reflect.FuncOf(nil, nil, false))
	}
	_ = initVars

	for _, dec := range pkg.Declarations {
		switch n := dec.(type) {
		case *ast.Var:
			if len(n.Identifiers) == len(n.Values) {
				panic("TODO(Gianluca): not implemented")
				// backupBuilder := c.fb
				// c.fb = initVars.Builder()
				// for i := range n.Identifiers {
				// 	c.compileVarsGetValue([]ast.Expression{n.Identifiers[i]}, n.Values[i], true)
				// }
				// c.fb = backupBuilder
			} else {
				panic("TODO(Gianluca): not implemented")
			}
			// TODO (Gianluca): this makes a new init function for every
			// variable, which is wrong. Putting initFn declaration
			// outside this switch is wrong too: init.-1 cannot be created
			// if there's no need.
			// initFn, _ := c.currentPkg.NewFunction("init.-1", reflect.FuncOf(nil, nil, false))
			// initBuilder := initFn.Builder()
			// if len(n.Identifiers) == 1 && len(n.Values) == 1 {
			// 	currentBuilder := c.fb
			// 	c.fb = initBuilder
			// 	reg := c.fb.NewRegister(reflect.Int)
			// 	c.compileExpr(n.Values[0], reg, reflect.Int)
			// 	c.fb = currentBuilder
			// 	name := "A"                 // TODO
			// 	v := interface{}(int64(10)) // TODO
			// 	c.currentPkg.DefineVariable(name, v)
			// } else {
			// 	panic("TODO: not implemented")
			// }
		case *ast.Func:
			fn := NewScrigoFunction("main", n.Ident.Name, n.Type.Reflect)
			c.availableScrigoFunctions[n.Ident.Name] = fn
			c.currentFunction = fn
			c.fb = fn.Builder()
			c.fb.EnterScope()
			c.prepareFunctionBodyParameters(n)
			addExplicitReturn(n)
			c.compileNodes(n.Body.Nodes)
			c.fb.End()
			c.fb.ExitScope()
		case *ast.Import:
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
						v := NewVariable(parserGoPkg.Name, ident, value)
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
				panic("TODO(Gianluca): not implemented")
			}
		}
	}
}

// quickCompileExpr checks if expr is a value or a register, putting it into
// out. If it's neither of them, both isValue and isRegister are false and
// content of out is unspecified.
// If expectedKind if different from evaluated kind, quick-compiling expression
// is impossibile so isValue and isRegister are both false and content of out is
// unspecified.
// TODO (Gianluca): add to function documentation:
// TODO (Gianluca): quickCompileExpr must evaluate only expression which does
// not need extra registers for evaluation.
func (c *Compiler) quickCompileExpr(expr ast.Expression, expectedKind reflect.Kind) (out int8, isValue, isRegister bool) {
	if kindToType(expectedKind) != kindToType(c.typeinfo[expr].Type.Kind()) {
		return 0, false, false
	}
	switch expr := expr.(type) {
	case *ast.Int: // TODO (Gianluca): must be removed, is here because of a type-checker's bug.
		return int8(expr.Value.Int64()), true, false
	case *ast.Identifier:
		if c.fb.IsVariable(expr.Name) {
			return c.fb.ScopeLookup(expr.Name), false, true
		}
		return 0, false, false
	case *ast.Value:
		kind := c.typeinfo[expr].Type.Kind()
		switch kind {
		case reflect.Bool:
			v := int8(0)
			if expr.Val.(bool) {
				v = 1
			}
			return v, true, false
		case reflect.Int:
			n := expr.Val.(int)
			if n < 0 || n > 127 {
				c := c.fb.MakeIntConstant(int64(n))
				return c, false, true
			} else {
				return int8(n), true, false
			}
		case reflect.Int64:
			n := expr.Val.(int64)
			if n < 0 || n > 127 {
				c := c.fb.MakeIntConstant(int64(n))
				return c, false, true
			} else {
				return int8(n), true, false
			}
		case reflect.Float64:
			// TODO (Gianluca): handle all kinds of floats.
			v := int8(expr.Val.(float64))
			return v, true, false

		case reflect.Int8,
			reflect.Int16,
			reflect.Int32,
			reflect.Uint,
			reflect.Uint8,
			reflect.Uint16,
			reflect.Uint32,
			reflect.Uint64,
			reflect.Uintptr,
			reflect.Float32,
			reflect.Complex64,
			reflect.Complex128,
			reflect.Array,
			reflect.Chan,
			reflect.Interface,
			reflect.Struct,
			reflect.UnsafePointer:
			panic(fmt.Sprintf("TODO: not implemented kind %q", kind))
		case reflect.String:
			sConst := c.fb.MakeStringConstant(expr.Val.(string))
			reg := c.fb.NewRegister(reflect.String)
			c.fb.Move(true, sConst, reg, reflect.String, reflect.String)
			return reg, false, true
		}
	case *ast.String: // TODO (Gianluca): must be removed, is here because of a type-checker's bug.
		sConst := c.fb.MakeStringConstant(expr.Text)
		reg := c.fb.NewRegister(reflect.String)
		c.fb.Move(true, sConst, reg, reflect.String, reflect.String)
		return reg, false, true
	case *ast.Func:
		typ := c.typeinfo[expr].Type
		reg := c.fb.NewRegister(reflect.Func)
		scrigoFunc := c.fb.Func(reg, typ)
		funcLitBuilder := scrigoFunc.Builder()
		currentFb := c.fb
		c.fb = funcLitBuilder
		c.fb.EnterScope()
		c.prepareFunctionBodyParameters(expr)
		addExplicitReturn(expr)
		c.compileNodes(expr.Body.Nodes)
		c.fb.ExitScope()
		c.fb = currentFb
		return reg, false, true
	}
	return 0, false, false
}

// prepareCallParameters prepares parameters (out and in) for a function call of
// type funcType and arguments args. Returns the list of return registers and
// their respective kind.
func (c *Compiler) prepareCallParameters(funcType reflect.Type, args []ast.Expression, isNative bool) ([]int8, []reflect.Kind) {
	numOut := funcType.NumOut()
	numIn := funcType.NumIn()
	regs := make([]int8, numOut)
	kinds := make([]reflect.Kind, numOut)
	for i := 0; i < numOut; i++ {
		kind := funcType.Out(i).Kind()
		regs[i] = c.fb.NewRegister(kind)
		kinds[i] = kind
	}
	if funcType.IsVariadic() {
		for i := 0; i < numIn-1; i++ {
			kind := funcType.In(i).Kind()
			reg := c.fb.NewRegister(kind)
			c.compileExpr(args[i], reg, kind)
		}
		if varArgs := len(args) - (numIn - 1); varArgs > 0 {
			kind := funcType.In(numIn - 1).Elem().Kind()
			if isNative {
				for i := 0; i < varArgs; i++ {
					reg := c.fb.NewRegister(kind)
					c.compileExpr(args[i], reg, kind)
				}
			} else {
				sliceReg := int8(numIn)
				c.fb.MakeSlice(true, true, funcType.In(numIn-1), int8(varArgs), int8(varArgs), sliceReg)
				for i := 0; i < varArgs; i++ {
					tmpReg := c.fb.NewRegister(kind)
					c.compileExpr(args[i+numIn-1], tmpReg, kind)
					indexReg := c.fb.NewRegister(reflect.Int)
					c.fb.Move(true, int8(i), indexReg, reflect.Int, reflect.Int)
					c.fb.SetSlice(false, sliceReg, tmpReg, indexReg, kind)
				}
			}
		}
	} else { // No-variadic function.
		if numIn > 1 && len(args) == 1 { // f(g()), where f takes more than 1 argument.
			regs, kinds := c.compileCall(args[0].(*ast.Call))
			for i := range regs {
				dstKind := funcType.In(i).Kind()
				reg := c.fb.NewRegister(dstKind)
				c.fb.Move(false, regs[i], reg, kinds[i], dstKind)
			}
		} else {
			for i := 0; i < numIn; i++ {
				kind := funcType.In(i).Kind()
				reg := c.fb.NewRegister(kind)
				c.compileExpr(args[i], reg, kind)
			}
		}
	}
	return regs, kinds
}

// prepareFunctionBodyParameters prepares fun's parameters (out and int) before
// compiling its body.
func (c *Compiler) prepareFunctionBodyParameters(fun *ast.Func) {
	// Reserves space for return parameters.
	fillParametersTypes(fun.Type.Result)
	for _, res := range fun.Type.Result {
		resType := res.Type.(*ast.Value).Val.(reflect.Type)
		kind := resType.Kind()
		retReg := c.fb.NewRegister(kind)
		// TODO (Gianluca): add support for named return parameters.
		// Binding retReg to the name of the paramter should be enough.
		_ = retReg
	}
	// Binds function argument names to pre-allocated registers.
	fillParametersTypes(fun.Type.Parameters)
	for _, par := range fun.Type.Parameters {
		parType := par.Type.(*ast.Value).Val.(reflect.Type)
		kind := parType.Kind()
		argReg := c.fb.NewRegister(kind)
		c.fb.BindVarReg(par.Ident.Name, argReg)
	}
}

// compileCall compiles call, returning the list of registers (and their
// respective kind) within which return values are inserted.
func (c *Compiler) compileCall(call *ast.Call) (regs []int8, kinds []reflect.Kind) {
	stackShift := StackShift{
		int8(c.fb.numRegs[reflect.Int]),
		int8(c.fb.numRegs[reflect.Float64]),
		int8(c.fb.numRegs[reflect.String]),
		int8(c.fb.numRegs[reflect.Interface]),
	}
	if ident, ok := call.Func.(*ast.Identifier); ok {
		if !c.fb.IsVariable(ident.Name) {
			if fun, isScrigoFunc := c.availableScrigoFunctions[ident.Name]; isScrigoFunc {
				regs, kinds := c.prepareCallParameters(fun.typ, call.Args, false)
				index := c.scrigoFunctionIndex(fun)
				c.fb.Call(index, stackShift, call.Pos().Line)
				return regs, kinds
			}
			if _, isNativeFunc := c.availableNativeFunctions[ident.Name]; isNativeFunc {
				fun := c.availableNativeFunctions[ident.Name]
				funcType := reflect.TypeOf(fun.fast)
				regs, kinds := c.prepareCallParameters(funcType, call.Args, true)
				index := c.nativeFunctionIndex(fun)
				if funcType.IsVariadic() {
					numVar := len(call.Args) - (funcType.NumIn() - 1)
					c.fb.CallNative(index, int8(numVar), stackShift)
				} else {
					c.fb.CallNative(index, NoVariadic, stackShift)
				}
				return regs, kinds
			}
		}
	}
	if sel, ok := call.Func.(*ast.Selector); ok {
		if name, ok := sel.Expr.(*ast.Identifier); ok {
			if isGoPkg := c.isNativePkg[name.Name]; isGoPkg {
				fun := c.availableNativeFunctions[name.Name+"."+sel.Ident]
				funcType := reflect.TypeOf(fun.fast)
				regs, kinds := c.prepareCallParameters(funcType, call.Args, true)
				index := c.nativeFunctionIndex(fun)
				if funcType.IsVariadic() {
					numVar := len(call.Args) - (funcType.NumIn() - 1)
					c.fb.CallNative(index, int8(numVar), stackShift)
				} else {
					c.fb.CallNative(index, NoVariadic, stackShift)
				}
				return regs, kinds
			} else {
				panic("TODO(Gianluca): not implemented")
			}
		}
	}
	var funReg int8
	out, _, isRegister := c.quickCompileExpr(call.Func, reflect.Func)
	if isRegister {
		funReg = out
	} else {
		funReg = c.fb.NewRegister(reflect.Func)
		c.compileExpr(call.Func, funReg, reflect.Func)
	}
	funcType := c.typeinfo[call.Func].Type
	regs, kinds = c.prepareCallParameters(funcType, call.Args, true)
	c.fb.CallIndirect(funReg, 0, stackShift)
	return regs, kinds
}

// compileExpr compiles expression expr and puts results into reg (of kind
// kind). Compiled result is discarded if reg is 0.
// TODO (Gianluca): review all "kind" arguments in every compileExpr call.
// TODO (Gianluca): use "tmpReg" instead "reg" and move evaluated value to reg only if reg != 0.
func (c *Compiler) compileExpr(expr ast.Expression, reg int8, dstKind reflect.Kind) {
	switch expr := expr.(type) {

	case *ast.BinaryOperator:
		if op := expr.Operator(); op == ast.OperatorAnd || op == ast.OperatorOr {
			cmp := int8(0)
			if op == ast.OperatorAnd {
				cmp = 1
			}
			c.fb.EnterStack()
			c.compileExpr(expr.Expr1, reg, dstKind)
			endIf := c.fb.NewLabel()
			c.fb.If(true, reg, ConditionEqual, cmp, reflect.Int)
			c.fb.Goto(endIf)
			c.compileExpr(expr.Expr2, reg, dstKind)
			c.fb.SetLabelAddr(endIf)
			c.fb.ExitStack()
			return
		}
		c.fb.EnterStack()
		kind := c.typeinfo[expr.Expr1].Type.Kind()
		op1 := c.fb.NewRegister(kind)
		c.compileExpr(expr.Expr1, op1, kind)
		var op2 int8
		var ky bool
		{
			out, isValue, isRegister := c.quickCompileExpr(expr.Expr2, kind)
			if isValue {
				op2 = out
				ky = true
			} else if isRegister {
				op2 = out
			} else {
				op2 = c.fb.NewRegister(kind)
				c.compileExpr(expr.Expr2, op2, kind)
			}
		}
		switch op := expr.Operator(); {
		case op == ast.OperatorAddition && kind == reflect.String:
			if reg != 0 {
				c.fb.Concat(op1, op2, reg)
			}
		case op == ast.OperatorAddition:
			if reg != 0 {
				c.fb.Add(ky, op1, op2, reg, kind)
			}
		case op == ast.OperatorSubtraction:
			if reg != 0 {
				c.fb.Sub(ky, op1, op2, reg, kind)
			}
		case op == ast.OperatorMultiplication:
			if reg != 0 {
				c.fb.Mul(op1, op2, reg, kind)
			}
		case op == ast.OperatorDivision:
			if reg != 0 {
				c.fb.Div(op1, op2, reg, kind)
			}
		case op == ast.OperatorModulo:
			if reg != 0 {
				c.fb.Rem(op1, op2, reg, kind)
			}
		case ast.OperatorEqual <= op && op <= ast.OperatorGreaterOrEqual:
			var cond Condition
			switch op {
			case ast.OperatorEqual:
				cond = ConditionEqual
			case ast.OperatorNotEqual:
				cond = ConditionNotEqual
			case ast.OperatorLess:
				cond = ConditionLess
			case ast.OperatorLessOrEqual:
				cond = ConditionLessOrEqual
			case ast.OperatorGreater:
				cond = ConditionGreater
			case ast.OperatorGreaterOrEqual:
				cond = ConditionGreaterOrEqual
			}
			if reg != 0 {
				c.fb.Move(true, 1, reg, kind, kind)
				c.fb.If(ky, op1, cond, op2, kind)
				c.fb.Move(true, 0, reg, kind, kind)
			}
		}
		c.fb.ExitStack()

	case *ast.Call:
		// Builtin call.
		c.fb.EnterStack()
		if c.typeinfo[expr.Func].IsBuiltin() {
			c.compileBuiltin(expr, reg, dstKind)
			c.fb.ExitStack()
			return
		}
		// Conversion.
		if val, ok := expr.Func.(*ast.Value); ok {
			if typ, ok := val.Val.(reflect.Type); ok {
				kind := c.typeinfo[expr.Args[0]].Type.Kind()
				arg := c.fb.NewRegister(kind)
				c.compileExpr(expr.Args[0], arg, kind)
				c.fb.Convert(arg, typ, reg)
				c.fb.ExitStack()
				return
			}
		}
		regs, kinds := c.compileCall(expr)
		if reg != 0 {
			c.fb.Move(false, regs[0], reg, kinds[0], kinds[0])
		}
		c.fb.ExitStack()

	case *ast.CompositeLiteral:
		typ := expr.Type.(*ast.Value).Val.(reflect.Type)
		switch typ.Kind() {
		case reflect.Slice, reflect.Array:
			size := int8(compositeLiteralLen(expr))
			// TODO(Gianluca): incorrect when reg is 0: slice is not
			// created, but values must be evaluated anyway.
			if reg != 0 {
				if typ.Kind() == reflect.Array {
					typ = reflect.SliceOf(typ.Elem())
				}
				c.fb.MakeSlice(true, true, typ, size, size, reg)
			}
			var index int8 = -1
			for _, kv := range expr.KeyValues {
				if kv.Key != nil {
					index = int8(kv.Key.(*ast.Value).Val.(int))
				} else {
					index++
				}
				c.fb.EnterStack()
				indexReg := c.fb.NewRegister(reflect.Int)
				c.fb.Move(true, index, indexReg, reflect.Int, reflect.Int)
				var kvalue bool
				var value int8
				out, isValue, isRegister := c.quickCompileExpr(kv.Value, typ.Elem().Kind())
				if isValue {
					value = out
					kvalue = true
				} else if isRegister {
					value = out
				} else {
					value = c.fb.NewRegister(typ.Elem().Kind())
					c.compileExpr(kv.Value, value, typ.Elem().Kind())
				}
				if reg != 0 {
					c.fb.SetSlice(kvalue, reg, value, indexReg, typ.Elem().Kind())
				}
				c.fb.ExitStack()
			}
		case reflect.Struct:
			panic("TODO: not implemented")
		case reflect.Map:
			// TODO (Gianluca): handle maps with bigger size.
			size := len(expr.KeyValues)
			sizeReg := c.fb.MakeIntConstant(int64(size))
			regType := c.fb.Type(typ)
			c.fb.MakeMap(regType, true, sizeReg, reg)
			for _, kv := range expr.KeyValues {
				keyReg := c.fb.NewRegister(typ.Key().Kind())
				valueReg := c.fb.NewRegister(typ.Elem().Kind())
				c.fb.EnterStack()
				c.compileExpr(kv.Key, keyReg, typ.Key().Kind())
				kValue := false // TODO(Gianluca).
				c.compileExpr(kv.Value, valueReg, typ.Elem().Kind())
				c.fb.ExitStack()
				c.fb.SetMap(kValue, reg, valueReg, keyReg)
			}
		}

	case *ast.TypeAssertion:
		kind := c.typeinfo[expr.Expr].Type.Kind()
		out, _, isRegister := c.quickCompileExpr(expr.Expr, kind)
		var exprReg int8
		if isRegister {
			exprReg = out
		} else {
			exprReg = c.fb.NewRegister(kind)
			c.compileExpr(expr.Expr, exprReg, kind)
		}
		typ := expr.Type.(*ast.Value).Val.(reflect.Type)
		c.fb.Assert(exprReg, typ, reg)
		c.fb.Nop()

	case *ast.Selector:
		if v, ok := c.availableVariables[expr.Expr.(*ast.Identifier).Name+"."+expr.Ident]; ok {
			if reg == 0 {
				return
			}
			index := c.variableIndex(v)
			c.fb.GetVar(index, reg) // TODO (Gianluca): to review.
			return
		}
		if nf, ok := c.availableNativeFunctions[expr.Expr.(*ast.Identifier).Name+"."+expr.Ident]; ok {
			if reg == 0 {
				return
			}
			index := c.nativeFunctionIndex(nf)
			c.fb.GetFunc(true, index, reg)
			return
		}
		if sf, ok := c.availableScrigoFunctions[expr.Expr.(*ast.Identifier).Name+"."+expr.Ident]; ok {
			if reg == 0 {
				return
			}
			index := c.scrigoFunctionIndex(sf)
			c.fb.GetFunc(false, index, reg)
			return
		}
		exprKind := c.typeinfo[expr.Expr].Type.Kind()
		exprReg := c.fb.NewRegister(exprKind)
		c.compileExpr(expr.Expr, exprReg, exprKind)
		field := int8(0) // TODO(Gianluca).
		c.fb.Selector(exprReg, field, reg)

	case *ast.UnaryOperator:
		kind := c.typeinfo[expr.Expr].Type.Kind()
		tmpReg := c.fb.NewRegister(kind)
		c.compileExpr(expr.Expr, tmpReg, dstKind)
		switch expr.Operator() {
		case ast.OperatorNot:
			c.fb.SubInv(true, tmpReg, int8(1), tmpReg, reflect.Int)
			if reg != 0 {
				c.fb.Move(false, tmpReg, reg, kind, kind)
			}
		case ast.OperatorAmpersand:
			panic("TODO: not implemented")
		case ast.OperatorAddition:
			// Do nothing.
		case ast.OperatorSubtraction:
			c.fb.SubInv(true, tmpReg, 0, tmpReg, kind)
			if reg != 0 {
				c.fb.Move(false, tmpReg, reg, kind, kind)
			}
		case ast.OperatorMultiplication:
			panic("TODO: not implemented")
		}

	case *ast.Value:
		if reg == 0 {
			return
		}
		kind := c.typeinfo[expr].Type.Kind()
		out, isValue, isRegister := c.quickCompileExpr(expr, kind)
		if isValue {
			c.fb.Move(true, out, reg, kind, dstKind)
		} else if isRegister {
			c.fb.Move(false, out, reg, kind, dstKind)
		} else {
			kind := c.typeinfo[expr].Type.Kind()
			switch kind {
			case reflect.Slice, reflect.Map, reflect.Func, reflect.Ptr:
				genConst := c.fb.MakeGeneralConstant(expr.Val)
				c.fb.Move(true, genConst, reg, reflect.Interface, reflect.Interface)
			default:
				panic("bug")
			}
		}

	case *ast.Identifier:
		if reg == 0 {
			return
		}
		kind := c.typeinfo[expr].Type.Kind()
		out, isValue, isRegister := c.quickCompileExpr(expr, kind)
		if isValue {
			c.fb.Move(true, out, reg, kind, dstKind)
		} else if isRegister {
			c.fb.Move(false, out, reg, kind, dstKind)
		} else {
			if fun, isScrigoFunc := c.availableScrigoFunctions[expr.Name]; isScrigoFunc {
				index := c.scrigoFunctionIndex(fun)
				c.fb.GetFunc(false, index, reg)
			} else {
				panic("bug")
			}
		}

	case *ast.Int, *ast.String: // TODO (Gianluca): remove Int and String
		if reg == 0 {
			return
		}
		kind := c.typeinfo[expr].Type.Kind()
		out, isValue, isRegister := c.quickCompileExpr(expr, kind)
		if isValue {
			c.fb.Move(true, out, reg, kind, dstKind)
		} else if isRegister {
			c.fb.Move(false, out, reg, kind, dstKind)
		} else {
			panic("bug")
		}

	case *ast.Index:
		exprType := c.typeinfo[expr.Expr].Type
		var exprReg int8
		out, _, isRegister := c.quickCompileExpr(expr.Expr, exprType.Kind())
		if isRegister {
			exprReg = out
		} else {
			exprReg = c.fb.NewRegister(exprType.Kind())
		}
		out, isValue, isRegister := c.quickCompileExpr(expr.Index, reflect.Int)
		ki := false
		var i int8
		if isValue {
			ki = true
			i = out
		} else if isRegister {
			i = out
		} else {
			i = c.fb.NewRegister(reflect.Int)
			c.compileExpr(expr.Index, i, dstKind)
		}
		c.fb.Index(ki, exprReg, i, reg, exprType)

	case *ast.Slicing:
		panic("TODO: not implemented")

	case *ast.Rune, *ast.Float: // TODO (Gianluca): to review.
		panic("bug")

	default:
		panic(fmt.Sprintf("compileExpr currently does not support %T nodes", expr))

	}

}

// compileAssignment assign value to variables. If variables contains more than
// one variable, value must be a function call, a map indexing operation or a
// type assertion. These last two cases involve that variables contains 2
// elements.
// TODO (Gianluca): in case of variable declaration, if quickCompileExpr returns
// a register use it instead of creating a new one.
func (c *Compiler) compileAssignment(variables []ast.Expression, value ast.Expression, isDecl bool) {
	if len(variables) == 1 {
		variable := variables[0]
		kind := c.typeinfo[value].Type.Kind()
		if isBlankIdentifier(variable) {
			c.compileNodes([]ast.Node{value})
			return
		}
		switch variable := variable.(type) {
		case *ast.Selector:
			if v, ok := c.availableVariables[variable.Expr.(*ast.Identifier).Name+"."+variable.Ident]; ok {
				index := c.variableIndex(v)
				out, _, isRegister := c.quickCompileExpr(value, kind)
				var tmpReg int8
				if isRegister {
					tmpReg = out
				} else {
					tmpReg = c.fb.NewRegister(kind)
					c.compileExpr(value, tmpReg, kind)
				}
				c.fb.SetVar(tmpReg, index)
				return
			}
			panic("TODO: not implemented")
		case *ast.Identifier:
			var varReg int8
			if isDecl {
				varReg = c.fb.NewRegister(kind)
				c.fb.BindVarReg(variable.Name, varReg)
			} else {
				varReg = c.fb.ScopeLookup(variable.Name)
			}
			out, isValue, isRegister := c.quickCompileExpr(value, kind)
			if isValue {
				c.fb.Move(true, out, varReg, kind, kind)
			} else if isRegister {
				c.fb.Move(false, out, varReg, kind, kind)
			} else {
				c.fb.EnterStack()
				tmpReg := c.fb.NewRegister(kind)
				c.compileExpr(value, tmpReg, kind)
				c.fb.Move(false, tmpReg, varReg, kind, kind)
				c.fb.ExitStack()
			}
		case *ast.Index:
			switch exprType := c.typeinfo[variable.Expr].Type; exprType.Kind() {
			case reflect.Slice:
				var slice int8
				out, _, isRegister := c.quickCompileExpr(variable.Expr, reflect.Interface)
				if isRegister {
					slice = out
				} else {
					slice = c.fb.NewRegister(reflect.Interface)
					c.compileExpr(variable.Expr, slice, kind)
				}
				var kvalue bool
				var valueReg int8
				out, isValue, isRegister := c.quickCompileExpr(value, exprType.Elem().Kind())
				if isValue {
					valueReg = out
					kvalue = true
				} else if isRegister {
					valueReg = out
				} else {
					valueReg = c.fb.NewRegister(kind)
					c.compileExpr(value, valueReg, kind)
				}
				var index int8
				out, _, isRegister = c.quickCompileExpr(variable.Index, reflect.Int)
				if isRegister {
					index = out
				} else {
					index = c.fb.NewRegister(reflect.Int)
					c.compileExpr(variable.Index, index, reflect.Int)
				}
				c.fb.SetSlice(kvalue, slice, valueReg, index, kind)
			default:
				panic("TODO: not implemented")
			}
		default:
			panic("TODO: not implemented")
		}
		return
	}
	// len(variables) > 1
	// TODO (Gianluca): handle "_".
	switch value := value.(type) {
	case *ast.Call:
		varRegs := []int8{}
		for i := 0; i < len(variables); i++ {
			variable := variables[i]
			kind := c.typeinfo[variable].Type.Kind()
			var varReg int8
			if isDecl {
				varReg = c.fb.NewRegister(kind)
				c.fb.BindVarReg(variable.(*ast.Identifier).Name, varReg)
			} else {
				varReg = c.fb.ScopeLookup(variable.(*ast.Identifier).Name)
			}
			varRegs = append(varRegs, varReg)
		}
		retRegs, retKinds := c.compileCall(value)
		for i := range retRegs {
			c.fb.Move(false, retRegs[i], varRegs[i], retKinds[i], retKinds[i])
		}
	case *ast.TypeAssertion:
		var dst int8
		if isDecl {
			kind := c.typeinfo[variables[0]].Type.Kind()
			dst = c.fb.NewRegister(kind)
			c.fb.BindVarReg(variables[0].(*ast.Identifier).Name, dst)
		} else {
			dst = c.fb.ScopeLookup(variables[0].(*ast.Identifier).Name)
		}
		var okReg int8
		if isDecl {
			okReg = c.fb.NewRegister(reflect.Int)
			c.fb.BindVarReg(variables[1].(*ast.Identifier).Name, okReg)
		} else {
			okReg = c.fb.ScopeLookup(variables[1].(*ast.Identifier).Name)
		}
		typ := value.Type.(*ast.Value).Val.(reflect.Type)
		kind := c.typeinfo[value.Expr].Type.Kind()
		expr := c.fb.NewRegister(kind)
		c.compileExpr(value.Expr, expr, kind)
		c.fb.Assert(expr, typ, dst)
		c.fb.Move(true, 0, okReg, reflect.Int, reflect.Int)
		c.fb.Move(true, 1, okReg, reflect.Int, reflect.Int)
	default:
		panic("bug")
	}
}

// TODO (Gianluca): a builtin can be shadowed, but the compiler can't know it.
// Typechecker should flag *ast.Call nodes with a boolean indicating if it's a
// builtin.
func (c *Compiler) compileBuiltin(call *ast.Call, reg int8, dstKind reflect.Kind) {
	switch call.Func.(*ast.Identifier).Name {
	case "append":
		panic("TODO: not implemented")
	case "cap":
		panic("TODO: not implemented")
	case "close":
		panic("TODO: not implemented")
	case "complex":
		panic("TODO: not implemented")
	case "copy":
		var dst, src int8
		out, _, isRegister := c.quickCompileExpr(call.Args[0], reflect.Slice)
		if isRegister {
			dst = out
		} else {
			dst = c.fb.NewRegister(reflect.Slice)
			c.compileExpr(call.Args[0], dst, reflect.Slice)
		}
		out, _, isRegister = c.quickCompileExpr(call.Args[1], reflect.Slice)
		if isRegister {
			src = out
		} else {
			src = c.fb.NewRegister(reflect.Slice)
			c.compileExpr(call.Args[0], src, reflect.Slice)
		}
		c.fb.Copy(dst, src, reg)
	case "delete":
		mapExpr := call.Args[0]
		keyExpr := call.Args[1]
		mapType := c.typeinfo[mapExpr].Type
		keyType := c.typeinfo[keyExpr].Type
		var mapp, key int8
		out, _, isRegister := c.quickCompileExpr(mapExpr, reflect.Map)
		if isRegister {
			mapp = out
		} else {
			mapp = c.fb.NewRegister(mapType.Kind())
			c.compileExpr(mapExpr, mapp, mapType.Kind())
		}
		out, _, isRegister = c.quickCompileExpr(keyExpr, keyType.Kind())
		if isRegister {
			key = out
		} else {
			key = c.fb.NewRegister(keyType.Kind())
			c.compileExpr(keyExpr, key, keyType.Kind())
		}
		c.fb.Delete(mapp, key)
	case "imag":
		panic("TODO: not implemented")
	case "html": // TODO (Gianluca): to review.
		panic("TODO: not implemented")
	case "len":
		typ := c.typeinfo[call.Args[0]].Type
		s := c.fb.NewRegister(typ.Kind())
		c.compileExpr(call.Args[0], s, typ.Kind())
		c.fb.Len(s, reg, typ)
	case "make":
		typ := call.Args[0].(*ast.Value).Val.(reflect.Type)
		regType := c.fb.Type(typ)
		switch typ.Kind() {
		case reflect.Map:
			var size int8
			var kSize bool
			out, isValue, isRegister := c.quickCompileExpr(call.Args[1], reflect.Int)
			if isValue {
				kSize = true
				size = out
			} else if isRegister {
				size = out
			} else {
				size = c.fb.NewRegister(reflect.Int)
				c.compileExpr(call.Args[1], size, reflect.Int)
			}
			c.fb.MakeMap(regType, kSize, size, reg)
		case reflect.Slice:
			lenExpr := call.Args[1]
			capExpr := call.Args[2]
			var len, cap int8
			var kLen, kCap bool
			out, isValue, isRegister := c.quickCompileExpr(lenExpr, reflect.Int)
			if isValue {
				len = out
				kLen = true
			} else if isRegister {
				len = out
			} else {
				len = c.fb.NewRegister(reflect.Int)
				c.compileExpr(lenExpr, len, reflect.Int)
			}
			out, isValue, isRegister = c.quickCompileExpr(capExpr, reflect.Int)
			if isValue {
				cap = out
				kCap = true
			} else if isRegister {
				cap = out
			} else {
				cap = c.fb.NewRegister(reflect.Int)
				c.compileExpr(capExpr, cap, reflect.Int)
			}
			c.fb.MakeSlice(kLen, kCap, typ, len, cap, reg)
		case reflect.Chan:
			panic("TODO: not implemented")
		default:
			panic("bug")
		}
	case "new":
		panic("TODO: not implemented")
		// typ := c.typeinfo[call.Args[0]].Type
		// t := c.currFb.Type(typ)
		// i = instruction{op: opNew, b: t, c: }
	case "panic":
		arg := call.Args[0]
		var reg int8
		out, _, isRegister := c.quickCompileExpr(arg, reflect.Interface)
		if isRegister {
			reg = out
		} else {
			reg = c.fb.NewRegister(reflect.Interface)
			c.compileExpr(arg, reg, reflect.Interface)
		}
		c.fb.Panic(reg, call.Pos().Line)
	case "print":
		for i := range call.Args {
			arg := c.fb.NewRegister(reflect.Interface)
			c.compileExpr(call.Args[i], arg, reflect.Interface)
			c.fb.Print(arg)
		}
	case "println":
		panic("TODO: not implemented")
	case "real":
		panic("TODO: not implemented")
	case "recover":
		c.fb.Recover(reg)
	default:
		panic("unkown builtin") // TODO(Gianluca): remove.
	}
}

// compileNodes compiles nodes.
func (c *Compiler) compileNodes(nodes []ast.Node) {
	for _, node := range nodes {
		switch node := node.(type) {

		case *ast.Assignment:
			// TODO (Gianluca): clean up.
			switch {
			case len(node.Variables) == 1 && len(node.Values) == 1 && (node.Type == ast.AssignmentAddition || node.Type == ast.AssignmentSubtraction):
				ident := node.Variables[0].(*ast.Identifier)
				varReg := c.fb.ScopeLookup(ident.Name)
				kind := c.typeinfo[ident].Type.Kind()
				out, isValue, isRegister := c.quickCompileExpr(node.Values[0], kind)
				var y int8
				var ky bool
				if isValue {
					y = out
					ky = true
				} else if isRegister {
					y = out
				} else {
					c.compileExpr(node.Values[0], y, kind)
				}
				if node.Type == ast.AssignmentAddition {
					c.fb.Add(ky, varReg, y, varReg, kind)
				} else {
					c.fb.Sub(ky, varReg, y, varReg, kind)
				}
			case len(node.Variables) == len(node.Values):
				for i := range node.Variables {
					c.compileAssignment([]ast.Expression{node.Variables[i]}, node.Values[i], node.Type == ast.AssignmentDeclaration)
				}
			case len(node.Variables) > 1 && len(node.Values) == 1:
				c.compileAssignment(node.Variables, node.Values[0], node.Type == ast.AssignmentDeclaration)
			case len(node.Variables) == 1 && len(node.Values) == 0:
				switch node.Type {
				case ast.AssignmentIncrement:
					name := node.Variables[0].(*ast.Identifier).Name
					reg := c.fb.ScopeLookup(name)
					kind := c.typeinfo[node.Variables[0]].Type.Kind()
					c.fb.Add(true, reg, 1, reg, kind)
				case ast.AssignmentDecrement:
					panic("TODO: not implemented")
				default:
					panic("bug")
				}
			default:
				panic("bug")
			}

		case *ast.Block:
			c.fb.EnterScope()
			c.compileNodes(node.Nodes)
			c.fb.ExitScope()

		case *ast.Defer, *ast.Go:
			if def, ok := node.(*ast.Defer); ok {
				if c.typeinfo[def.Call.Func].IsBuiltin() {
					ident := def.Call.Func.(*ast.Identifier)
					if ident.Name == "recover" {
						continue
					} else {
						// TODO(Gianluca): builtins (except recover)
						// must be incapsulated inside a function
						// literal call when deferring (or starting
						// a goroutine?). For example
						//
						//	defer copy(dst, src)
						//
						// should be compiled into
						//
						// 	defer func() {
						// 		copy(dst, src)
						// 	}()
						//
						panic("TODO(Gianluca): not implemented")
					}
				}
			}
			funReg := c.fb.NewRegister(reflect.Func)
			var funNode ast.Expression
			var args []ast.Expression
			switch node := node.(type) {
			case *ast.Defer:
				funNode = node.Call.Func
				args = node.Call.Args
			case *ast.Go:
				funNode = node.Call.Func
				args = node.Call.Args
			}
			funType := c.typeinfo[funNode].Type
			c.compileExpr(funNode, funReg, reflect.Func)
			offset := StackShift{
				int8(c.fb.numRegs[reflect.Int]),
				int8(c.fb.numRegs[reflect.Float64]),
				int8(c.fb.numRegs[reflect.String]),
				int8(c.fb.numRegs[reflect.Interface]),
			}
			// TODO(Gianluca): currently supports only deferring or
			// starting goroutines of Scrigo defined functions.
			isNative := false
			c.prepareCallParameters(funType, args, isNative)
			// TODO(Gianluca): currently supports only deferring functions
			// and starting goroutines with no arguments and no return
			// parameters.
			argsShift := StackShift{}
			switch node.(type) {
			case *ast.Defer:
				c.fb.Defer(funReg, NoVariadic, offset, argsShift)
			case *ast.Go:
				// TODO(Gianluca):
				c.fb.Go()
			}

		case *ast.If:
			c.fb.EnterScope()
			if node.Assignment != nil {
				c.compileNodes([]ast.Node{node.Assignment})
			}
			x, y, kind, o, ky := c.compileCondition(node.Condition)
			c.fb.If(ky, x, o, y, kind)
			if node.Else == nil { // TODO (Gianluca): can "then" and "else" be unified in some way?
				endIfLabel := c.fb.NewLabel()
				c.fb.Goto(endIfLabel)
				c.compileNodes(node.Then.Nodes)
				c.fb.SetLabelAddr(endIfLabel)
			} else {
				elseLabel := c.fb.NewLabel()
				c.fb.Goto(elseLabel)
				c.compileNodes(node.Then.Nodes)
				endIfLabel := c.fb.NewLabel()
				c.fb.Goto(endIfLabel)
				c.fb.SetLabelAddr(elseLabel)
				if node.Else != nil {
					switch els := node.Else.(type) {
					case *ast.If:
						c.compileNodes([]ast.Node{els})
					case *ast.Block:
						c.compileNodes(els.Nodes)
					}
				}
				c.fb.SetLabelAddr(endIfLabel)
			}
			c.fb.ExitScope()

		case *ast.For:
			c.fb.EnterScope()
			if node.Init != nil {
				c.compileNodes([]ast.Node{node.Init})
			}
			if node.Condition != nil {
				forLabel := c.fb.NewLabel()
				c.fb.SetLabelAddr(forLabel)
				x, y, kind, o, ky := c.compileCondition(node.Condition)
				c.fb.If(ky, x, o, y, kind)
				endForLabel := c.fb.NewLabel()
				c.fb.Goto(endForLabel)
				if node.Post != nil {
					c.compileNodes([]ast.Node{node.Post})
				}
				c.compileNodes(node.Body)
				c.fb.Goto(forLabel)
				c.fb.SetLabelAddr(endForLabel)
			} else {
				forLabel := c.fb.NewLabel()
				c.fb.SetLabelAddr(forLabel)
				if node.Post != nil {
					c.compileNodes([]ast.Node{node.Post})
				}
				c.compileNodes(node.Body)
				c.fb.Goto(forLabel)
			}
			c.fb.ExitScope()

		case *ast.ForRange:
			panic("TODO: not implemented")
			// c.fb.EnterScope()
			// expr := c.fb.NewRegister(reflect.String)
			// kind := c.typeinfo[node.Assignment.Values[0]].Type.Kind()
			// c.compileExpr(node.Assignment.Values[0], expr)
			// c.fb.Range(expr, kind)
			// c.compileNodes(node.Body)
			// c.fb.ExitScope()

		case *ast.Return:
			if len(node.Values) == 1 {
				if _, isCall := node.Values[0].(*ast.Call); isCall {
					// TODO (Gianluca): must assign new values.
					// TODO (Gianluca): use the appropiate function, cause not necessarily is CurrentPackage, CurrentFunction.
					// c.fb.TailCall(CurrentPackage, CurrentFunction, node.Pos().Line)
					continue
				}
			}
			for i, v := range node.Values {
				kind := c.typeinfo[v].Type.Kind()
				reg := int8(i + 1)
				c.fb.allocRegister(kind, reg)
				c.compileExpr(v, reg, kind) // TODO (Gianluca): must be return parameter kind.
			}
			c.fb.Return()

		case *ast.Switch:
			c.compileSwitch(node)

		case *ast.TypeSwitch:
			c.compileTypeSwitch(node)

		case *ast.Var:
			if len(node.Identifiers) == len(node.Values) {
				for i := range node.Identifiers {
					c.compileAssignment([]ast.Expression{node.Identifiers[i]}, node.Values[i], true)
				}
			} else {
				expr := make([]ast.Expression, len(node.Identifiers))
				for i := range node.Identifiers {
					expr[i] = node.Identifiers[i]
				}
				c.compileAssignment(expr, node.Values[0], true)
			}

		case ast.Expression:
			// TODO (Gianluca): use 0 (which is no longer a valid
			// register) and handle it as a special case in compileExpr.
			c.compileExpr(node, 0, reflect.Invalid)

		}
	}
}

// compileTypeSwitch compiles type switch node.
// TODO (Gianluca): a type-checker bug does not replace type switch type with proper value.
func (c *Compiler) compileTypeSwitch(node *ast.TypeSwitch) {

	if node.Init != nil {
		c.compileNodes([]ast.Node{node.Init})
	}

	typAss := node.Assignment.Values[0].(*ast.TypeAssertion)
	kind := c.typeinfo[typAss.Expr].Type.Kind()
	expr := c.fb.NewRegister(kind)
	c.compileExpr(typAss.Expr, expr, kind)

	// typ := node.Assignment.Values[0].(*ast.Value).Val.(reflect.Type)
	// typReg := c.fb.Type(typ)
	// if variab := node.Assignment.Variables[0]; !isBlankIdentifier(variab) {
	// 	c.compileVarsGetValue([]ast.Expression{variab}, node.Assignment.Values[0], node.Assignment.Type == ast.AssignmentDeclaration)
	// }

	bodyLabels := make([]uint32, len(node.Cases))
	endSwitchLabel := c.fb.NewLabel()

	var defaultLabel uint32
	hasDefault := false

	for i, cas := range node.Cases {
		bodyLabels[i] = c.fb.NewLabel()
		hasDefault = hasDefault || cas.Expressions == nil
		for _, caseExpr := range cas.Expressions {
			caseType := caseExpr.(*ast.Value).Val.(reflect.Type)
			c.fb.Assert(expr, caseType, 0)
			next := c.fb.NewLabel()
			c.fb.Goto(next)
			c.fb.Goto(bodyLabels[i])
			c.fb.SetLabelAddr(next)
		}
	}

	if hasDefault {
		defaultLabel = c.fb.NewLabel()
		c.fb.Goto(defaultLabel)
	} else {
		c.fb.Goto(endSwitchLabel)
	}

	for i, cas := range node.Cases {
		if cas.Expressions == nil {
			c.fb.SetLabelAddr(defaultLabel)
		}
		c.fb.SetLabelAddr(bodyLabels[i])
		c.compileNodes(cas.Body)
		c.fb.Goto(endSwitchLabel)
	}

	c.fb.SetLabelAddr(endSwitchLabel)
}

// compileSwitch compiles switch node.
func (c *Compiler) compileSwitch(node *ast.Switch) {

	if node.Init != nil {
		c.compileNodes([]ast.Node{node.Init})
	}

	kind := c.typeinfo[node.Expr].Type.Kind()
	expr := c.fb.NewRegister(kind)
	c.compileExpr(node.Expr, expr, kind)
	bodyLabels := make([]uint32, len(node.Cases))
	endSwitchLabel := c.fb.NewLabel()

	var defaultLabel uint32
	hasDefault := false

	for i, cas := range node.Cases {
		bodyLabels[i] = c.fb.NewLabel()
		hasDefault = hasDefault || cas.Expressions == nil
		for _, caseExpr := range cas.Expressions {
			var ky bool
			var y int8
			out, isValue, isRegister := c.quickCompileExpr(caseExpr, kind)
			if isValue {
				ky = true
				y = out
			} else if isRegister {
				y = out
			} else {
				c.fb.allocRegister(kind, y)
				c.compileExpr(caseExpr, y, kind)
			}
			c.fb.If(ky, expr, ConditionNotEqual, y, kind) // Condizione negata per poter concatenare gli if
			c.fb.Goto(bodyLabels[i])
		}
	}

	if hasDefault {
		defaultLabel = c.fb.NewLabel()
		c.fb.Goto(defaultLabel)
	} else {
		c.fb.Goto(endSwitchLabel)
	}

	for i, cas := range node.Cases {
		if cas.Expressions == nil {
			c.fb.SetLabelAddr(defaultLabel)
		}
		c.fb.SetLabelAddr(bodyLabels[i])
		c.compileNodes(cas.Body)
		if !cas.Fallthrough {
			c.fb.Goto(endSwitchLabel)
		}
	}

	c.fb.SetLabelAddr(endSwitchLabel)
}

// compileCondition compiles expr using the current function builder. Returns
// the two values of the condition (x and y), a kind, the condition ad a boolean
// ky which indicates whether y is a constant value.
// TODO (Gianluca): implement missing conditions.
func (c *Compiler) compileCondition(expr ast.Expression) (x, y int8, kind reflect.Kind, o Condition, yk bool) {
	// ConditionEqual             Condition = iota // x == y
	// ConditionNotEqual                           // x != y
	// ConditionLess                               // x <  y
	// ConditionLessOrEqual                        // x <= y
	// ConditionGreater                            // x >  y
	// ConditionGreaterOrEqual                     // x >= y
	// ConditionEqualLen                           // len(x) == y
	// ConditionNotEqualLen                        // len(x) != y
	// ConditionLessLen                            // len(x) <  y
	// ConditionLessOrEqualLen                     // len(x) <= y
	// ConditionGreaterLen                         // len(x) >  y
	// ConditionGreaterOrEqualLen                  // len(x) >= y
	// ConditionNil                                // x == nil
	// ConditionNotNil                             // x != nil
	// ConditionOK                                 // [vm.ok]
	switch cond := expr.(type) {
	case *ast.BinaryOperator:
		kind = c.typeinfo[cond.Expr1].Type.Kind()
		var out int8
		var isValue, isRegister bool
		out, _, isRegister = c.quickCompileExpr(cond.Expr1, kind)
		if isRegister {
			x = out
		} else {
			x = c.fb.NewRegister(kind)
			c.compileExpr(cond.Expr1, x, kind)
		}
		if isNil(cond.Expr2) {
			switch cond.Operator() {
			case ast.OperatorEqual:
				o = ConditionNil
			case ast.OperatorNotEqual:
				o = ConditionNotNil
			}
		} else {
			out, isValue, isRegister = c.quickCompileExpr(cond.Expr2, kind)
			if isValue {
				y = out
				yk = true
			} else if isRegister {
				y = out
			} else {
				y = c.fb.NewRegister(kind)
				c.compileExpr(cond.Expr2, y, kind)
			}
			switch cond.Operator() {
			case ast.OperatorEqual:
				o = ConditionEqual
			case ast.OperatorGreater:
				o = ConditionGreater
			case ast.OperatorGreaterOrEqual:
				o = ConditionGreaterOrEqual
			case ast.OperatorLess:
				o = ConditionLess
			case ast.OperatorLessOrEqual:
				o = ConditionLessOrEqual
			case ast.OperatorNotEqual:
				o = ConditionNotEqual
			}
		}

	default:
		x := c.fb.NewRegister(kind)
		c.compileExpr(cond, x, kind)
		o = ConditionEqual
		y = c.fb.MakeIntConstant(1) // TODO.
	}
	return x, y, kind, o, yk
}
