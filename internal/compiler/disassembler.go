// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"scriggo/runtime"
)

func packageName(pkg string) string {
	i := strings.LastIndex(pkg, "/")
	return pkg[i+1:]
}

func Disassemble(main *runtime.Function, globals []Global) (assembler map[string]string, err error) {

	functionsByPkg := map[string]map[*runtime.Function]int{}
	importsByPkg := map[string]map[string]struct{}{}

	c := len(main.Functions)
	if c == 0 {
		c = 1
	}
	allFunctions := make([]*runtime.Function, 1, c)
	allFunctions[0] = main

	for i := 0; i < len(allFunctions); i++ {
		fn := allFunctions[i]
		if p, ok := functionsByPkg[fn.Pkg]; ok {
			p[fn] = fn.Line
		} else {
			functionsByPkg[fn.Pkg] = map[*runtime.Function]int{fn: fn.Line}
		}
		for _, sf := range fn.Functions {
			if sf.Pkg != fn.Pkg {
				if packages, ok := importsByPkg[fn.Pkg]; ok {
					packages[sf.Pkg] = struct{}{}
				} else {
					importsByPkg[fn.Pkg] = map[string]struct{}{sf.Pkg: {}}
				}
			}
		}
		for _, nf := range fn.Predefined {
			if packages, ok := importsByPkg[fn.Pkg]; ok {
				packages[nf.Pkg] = struct{}{}
			} else {
				importsByPkg[fn.Pkg] = map[string]struct{}{nf.Pkg: {}}
			}
		}
		allFunctions = append(allFunctions, fn.Functions...)
	}

	assembler = map[string]string{}

	var b bytes.Buffer

	for path, funcs := range functionsByPkg {

		_, _ = b.WriteString("\nPackage ")
		_, _ = b.WriteString(packageName(path))
		_, _ = b.WriteRune('\n')

		var packages []string
		if imports, ok := importsByPkg[path]; ok {
			for pkg := range imports {
				packages = append(packages, pkg)
			}
			sort.Slice(packages, func(i, j int) bool { return packages[i] < packages[j] })
			for _, pkg := range packages {
				_, _ = b.WriteString("\nImport ")
				_, _ = b.WriteString(strconv.Quote(pkg))
				_, _ = b.WriteRune('\n')
			}
			packages = packages[:]
		}

		functions := make([]*runtime.Function, 0, len(funcs))
		for fn := range funcs {
			functions = append(functions, fn)
		}
		sort.Slice(functions, func(i, j int) bool { return functions[i].Line < functions[i].Line })

		for _, fn := range functions {
			_, _ = b.WriteString("\nFunc ")
			_, _ = b.WriteString(fn.Name)
			disassembleFunction(&b, fn, globals, 0)
		}
		_, _ = b.WriteRune('\n')

		assembler[path] = b.String()

		b.Reset()

	}

	return assembler, nil
}

func DisassembleFunction(w io.Writer, fn *runtime.Function, globals []Global) (int64, error) {
	var b bytes.Buffer
	_, _ = fmt.Fprintf(&b, "Func %s", fn.Name)
	disassembleFunction(&b, fn, globals, 0)
	return b.WriteTo(w)
}

func disassembleFunction(w *bytes.Buffer, fn *runtime.Function, globals []Global, depth int) {
	indent := ""
	if depth > 0 {
		indent = strings.Repeat("\t", depth)
	}
	labelOf := map[runtime.Addr]runtime.Addr{}
	for _, in := range fn.Body {
		switch in.Op {
		case runtime.OpBreak, runtime.OpContinue, runtime.OpGoto:
			labelOf[runtime.Addr(decodeUint24(in.A, in.B, in.C))] = 0
		}
	}
	if len(labelOf) > 0 {
		addresses := make([]int, len(labelOf))
		i := 0
		for addr := range labelOf {
			addresses[i] = int(addr)
			i++
		}
		sort.Ints(addresses)
		for i, addr := range addresses {
			labelOf[runtime.Addr(addr)] = runtime.Addr(i) + 1
		}
	}

	// Print input parameters.
	_, _ = fmt.Fprintf(w, "(")
	if fn.Type.NumIn() > 0 {
		out := map[runtime.Type]int{runtime.TypeInt: 0, runtime.TypeFloat: 0, runtime.TypeString: 0, runtime.TypeGeneral: 0}
		for i := 0; i < fn.Type.NumOut(); i++ {
			out[kindToType(fn.Type.Out(i).Kind())]++
		}
		in := map[runtime.Type]int{runtime.TypeInt: 0, runtime.TypeFloat: 0, runtime.TypeString: 0, runtime.TypeGeneral: 0}
		for i := 0; i < fn.Type.NumIn(); i++ {
			if i > 0 {
				_, _ = fmt.Fprint(w, ", ")
			}
			typ := fn.Type.In(i)
			label := registerKindToLabel(reflectToRegisterKind(typ.Kind()))
			vmType := kindToType(fn.Type.In(i).Kind())
			in[vmType]++
			reg := out[vmType] + in[vmType]
			_, _ = fmt.Fprintf(w, "%s%d %s", label, reg, typ)
		}
	}
	_, _ = fmt.Fprint(w, ")")

	// Print output parameters.
	if fn.Type.NumOut() > 0 {
		out := map[runtime.Type]int{runtime.TypeInt: 0, runtime.TypeFloat: 0, runtime.TypeString: 0, runtime.TypeGeneral: 0}
		_, _ = fmt.Fprint(w, " (")
		for i := 0; i < fn.Type.NumOut(); i++ {
			if i > 0 {
				_, _ = fmt.Fprint(w, ", ")
			}
			typ := fn.Type.Out(i)
			label := registerKindToLabel(reflectToRegisterKind(typ.Kind()))
			vmType := kindToType(fn.Type.Out(i).Kind())
			out[vmType]++
			_, _ = fmt.Fprintf(w, "%s%d %s", label, out[vmType], fn.Type.Out(i))
		}
		_, _ = fmt.Fprint(w, ")")
	}

	_, _ = fmt.Fprint(w, "\n")
	_, _ = fmt.Fprintf(w, "%s\t; regs(%d,%d,%d,%d)\n", indent,
		fn.NumReg[runtime.TypeInt], fn.NumReg[runtime.TypeFloat], fn.NumReg[runtime.TypeString], fn.NumReg[runtime.TypeGeneral])
	instrNum := runtime.Addr(len(fn.Body))
	for addr := runtime.Addr(0); addr < instrNum; addr++ {
		if label, ok := labelOf[runtime.Addr(addr)]; ok {
			_, _ = fmt.Fprintf(w, "%s%d:", indent, label)
		}
		in := fn.Body[addr]
		switch in.Op {
		case runtime.OpBreak, runtime.OpContinue, runtime.OpGoto:
			label := labelOf[runtime.Addr(decodeUint24(in.A, in.B, in.C))]
			_, _ = fmt.Fprintf(w, "%s\t%s %d", indent, operationName[in.Op], label)
		default:
			_, _ = fmt.Fprintf(w, "%s\t%s", indent, disassembleInstruction(fn, globals, addr))
		}
		if in.Op == runtime.OpLoadFunc && fn.Functions[uint8(in.B)].Parent != nil { // function literal
			_, _ = fmt.Fprint(w, " ", disassembleOperand(fn, in.C, reflect.Interface, false), " func")
			disassembleFunction(w, fn.Functions[uint8(in.B)], globals, depth+1)
		} else {
			_, _ = fmt.Fprint(w, "\n")
		}
		switch in.Op {
		case runtime.OpCall, runtime.OpCallIndirect, runtime.OpCallPredefined, runtime.OpTailCall, runtime.OpSlice, runtime.OpStringSlice:
			addr += 1
		case runtime.OpDefer:
			addr += 2
		}
		if in.Op == runtime.OpMakeSlice && in.B > 0 {
			addr += 1
		}
	}
}

func DisassembleInstruction(w io.Writer, fn *runtime.Function, globals []Global, addr runtime.Addr) (int64, error) {
	n, err := io.WriteString(w, disassembleInstruction(fn, globals, addr))
	return int64(n), err
}

// getKind returns the kind of the given operand (which can be 'a', 'b' or 'c')
// of the instruction at the given address. If the information about the kind of
// the operands have not been added by the emitter/builder, then the
// reflect.Invalid kind is returned.
func getKind(operand rune, fn *runtime.Function, addr runtime.Addr) reflect.Kind {
	debugInfo, ok := fn.DebugInfo[addr]
	if !ok {
		return reflect.Invalid
	}
	switch operand {
	case 'a':
		return debugInfo.OperandKind[0]
	case 'b':
		return debugInfo.OperandKind[1]
	case 'c':
		return debugInfo.OperandKind[2]
	default:
		panic(fmt.Errorf("BUG: invalid operand %v", operand))
	}
}

func disassembleInstruction(fn *runtime.Function, globals []Global, addr runtime.Addr) string {
	in := fn.Body[addr]
	op, a, b, c := in.Op, in.A, in.B, in.C
	k := false
	if op < 0 {
		op = -op
		k = true
	}
	s := operationName[op]
	switch op {
	case runtime.OpAddInt64, runtime.OpAddInt8, runtime.OpAddInt16, runtime.OpAddInt32,
		runtime.OpAnd, runtime.OpAndNot, runtime.OpOr, runtime.OpXor,
		runtime.OpDivInt64, runtime.OpDivInt8, runtime.OpDivInt16, runtime.OpDivInt32, runtime.OpDivUint8, runtime.OpDivUint16, runtime.OpDivUint32, runtime.OpDivUint64,
		runtime.OpMulInt64, runtime.OpMulInt8, runtime.OpMulInt16, runtime.OpMulInt32,
		runtime.OpRemInt64, runtime.OpRemInt8, runtime.OpRemInt16, runtime.OpRemInt32, runtime.OpRemUint8, runtime.OpRemUint16, runtime.OpRemUint32, runtime.OpRemUint64,
		runtime.OpSubInt64, runtime.OpSubInt8, runtime.OpSubInt16, runtime.OpSubInt32,
		runtime.OpSubInvInt64, runtime.OpSubInvInt8, runtime.OpSubInvInt16, runtime.OpSubInvInt32,
		runtime.OpLeftShift64, runtime.OpLeftShift8, runtime.OpLeftShift16, runtime.OpLeftShift32,
		runtime.OpRightShift, runtime.OpRightShiftU:
		s += " " + disassembleOperand(fn, a, reflect.Int, false)
		s += " " + disassembleOperand(fn, b, reflect.Int, k)
		s += " " + disassembleOperand(fn, c, reflect.Int, false)
	case runtime.OpAddFloat32, runtime.OpAddFloat64, runtime.OpDivFloat32, runtime.OpDivFloat64,
		runtime.OpMulFloat32, runtime.OpMulFloat64,
		runtime.OpSubFloat32, runtime.OpSubFloat64, runtime.OpSubInvFloat32, runtime.OpSubInvFloat64:
		s += " " + disassembleOperand(fn, a, reflect.Float64, false)
		s += " " + disassembleOperand(fn, b, reflect.Float64, k)
		s += " " + disassembleOperand(fn, c, reflect.Float64, false)
	case runtime.OpAddr:
		s += " " + disassembleOperand(fn, a, reflect.Interface, false)
		s += " " + disassembleOperand(fn, b, reflect.Int, false)
		s += " " + disassembleOperand(fn, c, reflect.Interface, false)
	case runtime.OpAlloc:
		if k {
			s += " " + strconv.Itoa(int(decodeUint24(a, b, c)))
		} else {
			s += " *"
		}
	case runtime.OpAppend:
		s += " " + disassembleOperand(fn, a, getKind('a', fn, addr), false)
		s += " " + disassembleOperand(fn, b-1, getKind('b', fn, addr), false)
		s += " " + disassembleOperand(fn, c, reflect.Interface, false)
	case runtime.OpAppendSlice:
		s += " " + disassembleOperand(fn, a, reflect.Interface, false)
		s += " " + disassembleOperand(fn, c, reflect.Interface, false)
	case runtime.OpSend:
		s += " " + disassembleOperand(fn, a, getKind('a', fn, addr), false)
		s += " " + disassembleOperand(fn, c, reflect.Interface, false)
	case runtime.OpAssert:
		s += " " + disassembleOperand(fn, a, reflect.Interface, false)
		s += " " + fn.Types[b].String()
		t := fn.Types[int(uint(b))]
		var kind = reflectToRegisterKind(t.Kind())
		s += " " + disassembleOperand(fn, c, kind, false)
	case runtime.OpBreak, runtime.OpContinue, runtime.OpGoto:
		s += " " + strconv.Itoa(int(decodeUint24(a, b, c)))
	case runtime.OpCall, runtime.OpCallIndirect, runtime.OpCallPredefined, runtime.OpTailCall, runtime.OpDefer:
		if a != runtime.CurrentFunction {
			switch op {
			case runtime.OpCall, runtime.OpTailCall:
				sf := fn.Functions[uint8(a)]
				s += " " + packageName(sf.Pkg) + "." + sf.Name
			case runtime.OpCallIndirect:
				s += " " + "("
				s += disassembleOperand(fn, a, reflect.Interface, false)
				s += ")"
			case runtime.OpCallPredefined:
				nf := fn.Predefined[uint8(a)]
				s += " " + packageName(nf.Pkg) + "." + nf.Name
			case runtime.OpDefer:
				s += " " + disassembleOperand(fn, a, reflect.Interface, false)
			}
		}
		grow := fn.Body[addr+1]
		stackShift := runtime.StackShift{int8(grow.Op), grow.A, grow.B, grow.C}
		if c != runtime.NoVariadicArgs && (op == runtime.OpCallIndirect || op == runtime.OpCallPredefined || op == runtime.OpDefer) {
			s += " ..." + strconv.Itoa(int(c))
		}
		for i := 0; i < 4; i++ {
			s += " "
			if stackShift[i] == 0 {
				_, fn := funcNameType(fn, a, addr, op)
				if fn != nil && !funcHasParameterInRegister(fn, runtime.Type(i)) {
					s += "_"
					continue
				}
			}
			s += string("ifsg"[i])
			s += strconv.Itoa(int(stackShift[i] + 1))
		}
		s += "\t; " + disassembleFunctionCall(fn, a, addr, op, stackShift, c)
	case runtime.OpCap:
		s += " " + disassembleOperand(fn, a, reflect.Interface, false)
		s += " " + disassembleOperand(fn, c, reflect.Int, false)
	case runtime.OpCase:
		switch reflect.SelectDir(a) {
		case reflect.SelectSend:
			s += " Send " + disassembleOperand(fn, b, reflect.Int, k) + " " + disassembleOperand(fn, c, reflect.Interface, false)
		case reflect.SelectRecv:
			s += " Recv " + disassembleOperand(fn, b, reflect.Int, false) + " " + disassembleOperand(fn, c, reflect.Interface, false)
		default:
			s += " Default"
		}
	case runtime.OpClose, runtime.OpPanic, runtime.OpPrint:
		s += " " + disassembleOperand(fn, a, reflect.Interface, false)
	case runtime.OpComplex64, runtime.OpComplex128:
		s += " " + disassembleOperand(fn, a, reflect.Float64, false)
		s += " " + disassembleOperand(fn, b, reflect.Float64, false)
		s += " " + disassembleOperand(fn, c, reflect.Interface, false)
	case runtime.OpConcat:
		s += " " + disassembleOperand(fn, a, reflect.String, false)
		s += " " + disassembleOperand(fn, b, reflect.String, k)
		s += " " + disassembleOperand(fn, c, reflect.String, false)
	case runtime.OpConvert:
		s += " " + disassembleOperand(fn, a, reflect.Interface, false)
		typ := fn.Types[int(uint(b))]
		s += " " + typ.String()
		s += " " + disassembleOperand(fn, c, typ.Kind(), false)
	case runtime.OpConvertInt, runtime.OpConvertUint:
		s += " " + disassembleOperand(fn, a, reflect.Int, false)
		typ := fn.Types[int(uint(b))]
		s += " " + typ.String()
		s += " " + disassembleOperand(fn, c, reflect.Kind(typ.Kind()), false)
	case runtime.OpConvertFloat:
		s += " " + disassembleOperand(fn, a, reflect.Float64, false)
		typ := fn.Types[int(uint(b))]
		s += " " + typ.String()
		s += " " + disassembleOperand(fn, c, reflect.Kind(typ.Kind()), false)
	case runtime.OpConvertString:
		s += " " + disassembleOperand(fn, a, reflect.String, false)
		typ := fn.Types[int(uint(b))]
		s += " " + typ.String()
		s += " " + disassembleOperand(fn, c, reflect.Kind(typ.Kind()), false)
	case runtime.OpCopy:
		s += " " + disassembleOperand(fn, a, reflect.Interface, false)
		s += " " + disassembleOperand(fn, b, reflect.Int, false)
		s += " " + disassembleOperand(fn, c, reflect.Interface, false)
	case runtime.OpDelete:
		s += " " + disassembleOperand(fn, a, reflect.Interface, false)
		s += " " + disassembleOperand(fn, b, reflect.Interface, false)
	case runtime.OpIf:
		switch runtime.Condition(b) {
		case runtime.ConditionOK, runtime.ConditionNotOK:
			s += " " + conditionName[b]
		case runtime.ConditionEqual, runtime.ConditionNotEqual:
			s += " " + disassembleOperand(fn, a, reflect.Interface, false)
			s += " " + conditionName[b]
			s += " " + disassembleOperand(fn, c, reflect.Interface, k)
		default:
			s += " " + conditionName[b]
			s += " " + disassembleOperand(fn, a, reflect.Interface, false)
		}
	case runtime.OpIfInt:
		s += " " + disassembleOperand(fn, a, reflect.Int, false)
		s += " " + conditionName[b]
		if runtime.Condition(b) >= runtime.ConditionEqual {
			s += " " + disassembleOperand(fn, c, reflect.Int, k)
		}
	case runtime.OpIfFloat:
		s += " " + disassembleOperand(fn, a, reflect.Float64, false)
		s += " " + conditionName[b]
		s += " " + disassembleOperand(fn, c, reflect.Float64, k)
	case runtime.OpIfString:
		s += " " + disassembleOperand(fn, a, reflect.String, false)
		s += " " + conditionName[b]
		if runtime.Condition(b) < runtime.ConditionEqualLen {
			if k && c >= 0 {
				s += " " + strconv.Quote(string(c))
			} else {
				s += " " + disassembleOperand(fn, c, reflect.String, k)
			}
		} else {
			s += " " + disassembleOperand(fn, c, reflect.Int, k)
		}
	case runtime.OpField:
		s += " " + disassembleOperand(fn, a, reflect.Interface, false)
		s += " " + fmt.Sprintf("%v", decodeFieldIndex(fn.Constants.Int[b]))
		s += " " + disassembleOperand(fn, c, getKind('c', fn, addr), false)
	case runtime.OpGetVar:
		s += " " + disassembleVarRef(fn, globals, int16(int(a)<<8|int(uint8(b))))
		s += " " + disassembleOperand(fn, c, getKind('c', fn, addr), false)
	case runtime.OpGetVarAddr:
		s += " " + disassembleVarRef(fn, globals, int16(int(a)<<8|int(uint8(b))))
		s += " " + disassembleOperand(fn, c, reflect.Interface, false)
	case runtime.OpGo, runtime.OpReturn:
	case runtime.OpIndex:
		s += " " + disassembleOperand(fn, a, reflect.Interface, false)
		s += " " + disassembleOperand(fn, b, reflect.Int, k)
		s += " " + disassembleOperand(fn, c, getKind('c', fn, addr), false)
	case runtime.OpIndexString:
		s += " " + disassembleOperand(fn, a, reflect.String, false)
		s += " " + disassembleOperand(fn, b, reflect.Int, k)
		s += " " + disassembleOperand(fn, c, reflect.Int, false)
	case runtime.OpLen:
		s += " " + strconv.Itoa(int(a))
		if a == 0 {
			s += " " + disassembleOperand(fn, b, reflect.String, false)
		} else {
			s += " " + disassembleOperand(fn, b, reflect.Interface, false)
		}
		s += " " + disassembleOperand(fn, c, reflect.Int, false)
	case runtime.OpLoadData:
		s += " " + strconv.Itoa(int(decodeInt16(a, b)))
		s += " " + disassembleOperand(fn, c, reflect.Func, false)
	case runtime.OpLoadFunc:
		if a == 0 {
			f := fn.Functions[uint8(b)]
			if f.Parent != nil { // f is a function literal.
				s = "Func" // overwrite s.
			} else {
				s += " " + packageName(f.Pkg) + "." + f.Name
				s += " " + disassembleOperand(fn, c, reflect.Interface, false)
			}
		} else { // LoadFunc (predefined).
			f := fn.Predefined[uint8(b)]
			s += " " + packageName(f.Pkg) + "." + f.Name
			s += " " + disassembleOperand(fn, c, reflect.Interface, false)
		}
	case runtime.OpLoadNumber:
		if a == 0 {
			s += " int"
			s += " " + fmt.Sprintf("%d", fn.Constants.Int[uint8(b)])
			s += " " + disassembleOperand(fn, c, reflect.Int, false)
		} else {
			s += " float"
			s += " " + fmt.Sprintf("%f", fn.Constants.Float[uint8(b)])
			s += " " + disassembleOperand(fn, c, reflect.Float64, false)
		}
	case runtime.OpMakeChan:
		s += " " + fn.Types[int(uint(a))].String()
		s += " " + disassembleOperand(fn, b, reflect.Int, k)
		s += " " + disassembleOperand(fn, c, reflect.Interface, false)
	case runtime.OpMakeMap:
		s += " " + fn.Types[int(uint(a))].String()
		s += " " + disassembleOperand(fn, b, reflect.Int, k)
		s += " " + disassembleOperand(fn, c, reflect.Interface, false)
	case runtime.OpMakeSlice:
		s += " " + fn.Types[int(uint(a))].Elem().String()
		if b > 0 {
			next := fn.Body[addr+1]
			s += " " + disassembleOperand(fn, next.A, reflect.Int, (b&(1<<1)) != 0)
			s += " " + disassembleOperand(fn, next.B, reflect.Int, (b&(1<<2)) != 0)
		} else {
			s += " 0 0"
		}
		s += " " + disassembleOperand(fn, c, reflect.Interface, false)
	case runtime.OpMethodValue:
		s += " " + disassembleOperand(fn, a, reflect.Interface, false)
		s += " " + disassembleOperand(fn, b, reflect.String, true)
		s += " " + disassembleOperand(fn, c, reflect.Interface, false)
	case runtime.OpMove:
		switch runtime.Type(a) {
		case runtime.TypeInt:
			s += " " + disassembleOperand(fn, b, reflect.Int, k)
			s += " " + disassembleOperand(fn, c, reflect.Int, false)
		case runtime.TypeFloat:
			s += " " + disassembleOperand(fn, b, reflect.Float64, k)
			s += " " + disassembleOperand(fn, c, reflect.Float64, false)
		case runtime.TypeString:
			s += " " + disassembleOperand(fn, b, reflect.String, k)
			s += " " + disassembleOperand(fn, c, reflect.String, false)
		case runtime.TypeGeneral:
			s += " " + disassembleOperand(fn, b, reflect.Interface, k)
			s += " " + disassembleOperand(fn, c, reflect.Interface, false)
		}
	case runtime.OpNew:
		s += " " + fn.Types[int(uint(b))].String()
		s += " " + disassembleOperand(fn, c, reflect.Interface, false)
	case runtime.OpRange:
		s += " " + disassembleOperand(fn, a, reflect.Interface, false)
		s += " " + disassembleOperand(fn, b, reflect.Int, false)
		s += " " + disassembleOperand(fn, c, reflect.Int, false)
	case runtime.OpRangeString:
		s += " " + disassembleOperand(fn, a, reflect.String, k)
		s += " " + disassembleOperand(fn, b, reflect.Int, false)
		s += " " + disassembleOperand(fn, c, reflect.Int, false)
	case runtime.OpRealImag:
		s += " " + disassembleOperand(fn, a, reflect.Interface, k)
		s += " " + disassembleOperand(fn, b, reflect.Float64, false)
		s += " " + disassembleOperand(fn, c, reflect.Float64, false)
	case runtime.OpReceive:
		s += " " + disassembleOperand(fn, a, reflect.Interface, false)
		s += " " + disassembleOperand(fn, b, reflect.Bool, false)
		s += " " + disassembleOperand(fn, c, getKind('c', fn, addr), false)
	case runtime.OpRecover:
		if a > 0 {
			s += " down"
		}
		if c != 0 {
			s += " " + disassembleOperand(fn, c, reflect.Interface, false)
		}
	case runtime.OpSetField:
		s += " " + disassembleOperand(fn, a, getKind('a', fn, addr), k)
		s += " " + disassembleOperand(fn, b, reflect.Interface, false)
		s += " " + fmt.Sprintf("%v", decodeFieldIndex(fn.Constants.Int[c]))
	case runtime.OpSetMap:
		// fn, addr, 'a'
		s += " " + disassembleOperand(fn, a, getKind('a', fn, addr), k)
		s += " " + disassembleOperand(fn, b, reflect.Interface, false)
		s += " " + disassembleOperand(fn, c, getKind('c', fn, addr), false)
	case runtime.OpSetSlice:
		s += " " + disassembleOperand(fn, a, getKind('a', fn, addr), k)
		s += " " + disassembleOperand(fn, b, reflect.Interface, false)
		s += " " + disassembleOperand(fn, c, reflect.Int, false)
	case runtime.OpSetVar:
		s += " " + disassembleOperand(fn, a, getKind('a', fn, addr), op < 0)
		s += " " + disassembleVarRef(fn, globals, int16(int(b)<<8|int(uint8(c))))
	case runtime.OpSlice:
		khigh := b&2 != 0
		high := fn.Body[addr+1].B
		if khigh && high == -1 {
			khigh = false
			high = 0
		}
		kmax := b&4 != 0
		max := fn.Body[addr+1].C
		if kmax && max == -1 {
			kmax = false
			max = 0
		}
		s += " " + disassembleOperand(fn, a, reflect.Interface, false)
		s += " " + disassembleOperand(fn, fn.Body[addr+1].A, reflect.Int, b&1 != 0)
		s += " " + disassembleOperand(fn, high, reflect.Int, khigh)
		s += " " + disassembleOperand(fn, max, reflect.Int, kmax)
		s += " " + disassembleOperand(fn, c, reflect.Interface, false)
	case runtime.OpStringSlice:
		khigh := b&2 != 0
		high := fn.Body[addr+1].B
		if khigh && high == -1 {
			khigh = false
			high = 0
		}
		s += " " + disassembleOperand(fn, a, reflect.String, false)
		s += " " + disassembleOperand(fn, fn.Body[addr+1].A, reflect.Int, b&1 != 0)
		s += " " + disassembleOperand(fn, high, reflect.Int, khigh)
		s += " " + disassembleOperand(fn, c, reflect.String, false)
	case runtime.OpTypify:
		typ := fn.Types[int(uint(a))]
		s += " " + typ.String()
		s += " " + disassembleOperand(fn, b, reflectToRegisterKind(typ.Kind()), k)
		s += " " + disassembleOperand(fn, c, reflect.Interface, false)
	}
	return s
}

// funcNameType returns the name and the type of the specified function, if
// available. Only one of index and addr is meaningful, depending on the
// operation specified by op.
func funcNameType(fn *runtime.Function, index int8, addr runtime.Addr, op runtime.Operation) (string, reflect.Type) {
	switch op {
	case runtime.OpCall:
		typ := fn.Functions[index].Type
		name := fn.Functions[index].Name
		return name, typ
	case runtime.OpCallPredefined:
		name := fn.Predefined[index].Name
		typ := reflect.TypeOf(fn.Predefined[index].Func)
		return name, typ
	case runtime.OpCallIndirect, runtime.OpDefer:
		return "", fn.DebugInfo[addr].FuncType
	case runtime.OpTailCall:
		panic("BUG: not implemented") // TODO.
	default:
		panic("BUG")
	}
}

// funcHasParameterInRegister reports whether the given function has at least
// one parameter (input or output) that is stored into the given register type.
func funcHasParameterInRegister(fn reflect.Type, reg runtime.Type) bool {
	for i := 0; i < fn.NumIn(); i++ {
		if kindToType(fn.In(i).Kind()) == reg {
			return true
		}
	}
	for i := 0; i < fn.NumOut(); i++ {
		if kindToType(fn.Out(i).Kind()) == reg {
			return true
		}
	}
	return false
}

// disassembleFunctionCall disassemble a function call returning an
// human-readable string representing the call. The result of this function is
// used as a comment to the byte code.
func disassembleFunctionCall(fn *runtime.Function, index int8, addr runtime.Addr, op runtime.Operation, stackShift runtime.StackShift, variadic int8) string {
	name, typ := funcNameType(fn, index, addr, op)
	if typ == nil {
		return ""
	}
	if name == "$initvars" {
		return "package vars init"
	}
	print := func(t reflect.Type) string {
		str := ""
		switch kindToType(t.Kind()) {
		case runtime.TypeInt:
			stackShift[0]++
			str += fmt.Sprintf("i%d %v", stackShift[0], t)
		case runtime.TypeFloat:
			stackShift[1]++
			str += fmt.Sprintf("f%d %v", stackShift[1], t)
		case runtime.TypeString:
			stackShift[2]++
			str += fmt.Sprintf("s%d %v", stackShift[2], t)
		case runtime.TypeGeneral:
			stackShift[3]++
			str += fmt.Sprintf("g%d %v", stackShift[3], t)
		}
		return str
	}
	out := ""
	for i := 0; i < typ.NumOut(); i++ {
		out += print(typ.Out(i))
		if i < typ.NumOut()-1 {
			out += ", "
		}
	}
	in := ""
	for i := 0; i < typ.NumIn()-1; i++ {
		in += print(typ.In(i)) + ", "
	}
	if typ.NumIn()-1 >= 0 {
		if variadic == runtime.NoVariadicArgs || variadic == 0 {
			in += print(typ.In(typ.NumIn() - 1))
		} else {
			varType := typ.In(typ.NumIn() - 1).Elem()
			for i := int8(0); i < variadic; i++ {
				in += print(varType)
				if i < variadic-1 {
					in += ", "
				}
			}
		}
	}
	return fmt.Sprintf("func(%s) (%s)", in, out)
}

func disassembleVarRef(fn *runtime.Function, globals []Global, ref int16) string {
	depth := 0
	for ref >= 0 && fn.Parent != nil {
		ref = fn.VarRefs[ref]
		depth++
		fn = fn.Parent
	}
	if depth == 0 {
		v := globals[ref]
		return packageName(v.Pkg) + "." + v.Name
	}
	s := disassembleOperand(fn, -int8(ref), reflect.Interface, false)
	if depth > 0 {
		s += "@" + strconv.Itoa(depth)
	}
	return s
}

func reflectToRegisterKind(kind reflect.Kind) reflect.Kind {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return reflect.Int
	case reflect.Bool:
		return reflect.Bool
	case reflect.Float32, reflect.Float64:
		return reflect.Float64
	case reflect.String:
		return reflect.String
	default:
		return reflect.Interface
	}
}

func registerKindToLabel(kind reflect.Kind) string {
	switch kind {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return "i"
	case reflect.Float32, reflect.Float64:
		return "f"
	case reflect.String:
		return "s"
	case reflect.Invalid:
		return "?"
	default:
		return "g"
	}
}

func disassembleOperand(fn *runtime.Function, op int8, kind reflect.Kind, constant bool) string {
	if constant {
		switch {
		case reflect.Int <= kind && kind <= reflect.Uintptr:
			return strconv.Itoa(int(op))
		case kind == reflect.Float64:
			return strconv.FormatFloat(float64(op), 'f', -1, 64)
		case kind == reflect.Float32:
			return strconv.FormatFloat(float64(op), 'f', -1, 32)
		case kind == reflect.Bool:
			if op == 0 {
				return "false"
			}
			return "true"
		case kind == reflect.String:
			return strconv.Quote(fn.Constants.String[uint8(op)])
		case kind == reflect.Invalid:
			return "?"
		default:
			v := fn.Constants.General[uint8(op)]
			if v == nil {
				return "nil"
			}
			return fmt.Sprintf("%#v", v)
		}
	}
	label := registerKindToLabel(kind)
	if op > 0 {
		return label + strconv.Itoa(int(op))
	}
	if op == 0 {
		return "_"
	}
	return "(" + label + strconv.Itoa(-int(op)) + ")"
}

var operationName = [...]string{

	runtime.OpNone: "Nop",

	runtime.OpAddInt64:   "Add",
	runtime.OpAddInt8:    "Add8",
	runtime.OpAddInt16:   "Add16",
	runtime.OpAddInt32:   "Add32",
	runtime.OpAddFloat32: "Add32",
	runtime.OpAddFloat64: "Add",

	runtime.OpAddr: "Addr",

	runtime.OpAlloc: "Alloc",

	runtime.OpAnd: "And",

	runtime.OpAndNot: "AndNot",

	runtime.OpAppend: "Append",

	runtime.OpAppendSlice: "AppendSlice",

	runtime.OpAssert: "Assert",

	runtime.OpBreak: "Break",

	runtime.OpCall: "Call",

	runtime.OpCallIndirect: "Call",

	runtime.OpCallPredefined: "Call",

	runtime.OpCap: "Cap",

	runtime.OpCase: "Case",

	runtime.OpClose: "Close",

	runtime.OpComplex64:  "Complex64",
	runtime.OpComplex128: "Complex128",

	runtime.OpContinue: "Continue",

	runtime.OpConvert:       "Convert",
	runtime.OpConvertInt:    "Convert",
	runtime.OpConvertUint:   "ConvertU",
	runtime.OpConvertFloat:  "Convert",
	runtime.OpConvertString: "Convert",

	runtime.OpConcat: "Concat",

	runtime.OpCopy: "Copy",

	runtime.OpDefer: "Defer",

	runtime.OpDelete: "Delete",

	runtime.OpDivInt64:   "Div",
	runtime.OpDivInt8:    "Div8",
	runtime.OpDivInt16:   "Div16",
	runtime.OpDivInt32:   "Div32",
	runtime.OpDivUint8:   "DivU8",
	runtime.OpDivUint16:  "DivU16",
	runtime.OpDivUint32:  "DivU32",
	runtime.OpDivUint64:  "DivU64",
	runtime.OpDivFloat32: "Div32",
	runtime.OpDivFloat64: "Div",

	runtime.OpGetVar: "GetVar",

	runtime.OpGetVarAddr: "GetVarAddr",

	runtime.OpGo: "Go",

	runtime.OpGoto: "Goto",

	runtime.OpIf:       "If",
	runtime.OpIfInt:    "If",
	runtime.OpIfFloat:  "If",
	runtime.OpIfString: "If",

	runtime.OpIndex:       "Index",
	runtime.OpIndexString: "Index",

	runtime.OpLeftShift64: "LeftShift",
	runtime.OpLeftShift8:  "LeftShift8",
	runtime.OpLeftShift16: "LeftShift16",
	runtime.OpLeftShift32: "LeftShift32",

	runtime.OpLen: "Len",

	runtime.OpLoadData: "LoadData",

	runtime.OpLoadFunc: "LoadFunc",

	runtime.OpLoadNumber: "LoadNumber",

	runtime.OpMakeChan: "MakeChan",

	runtime.OpMakeMap: "MakeMap",

	runtime.OpMakeSlice: "MakeSlice",

	runtime.OpMapIndex: "MapIndex",

	runtime.OpMethodValue: "MethodValue",

	runtime.OpMove: "Move",

	runtime.OpMulInt64:   "Mul",
	runtime.OpMulInt8:    "Mul8",
	runtime.OpMulInt16:   "Mul16",
	runtime.OpMulInt32:   "Mul32",
	runtime.OpMulFloat32: "Mul32",
	runtime.OpMulFloat64: "Mul",

	runtime.OpNew: "New",

	runtime.OpOr: "Or",

	runtime.OpPanic: "Panic",

	runtime.OpPrint: "Print",

	runtime.OpRange: "Range",

	runtime.OpRangeString: "Range",

	runtime.OpRealImag: "RealImag",

	runtime.OpReceive: "Receive",

	runtime.OpRecover: "Recover",

	runtime.OpRemInt64:  "Rem",
	runtime.OpRemInt8:   "Rem8",
	runtime.OpRemInt16:  "Rem16",
	runtime.OpRemInt32:  "Rem32",
	runtime.OpRemUint8:  "RemU8",
	runtime.OpRemUint16: "RemU16",
	runtime.OpRemUint32: "RemU32",
	runtime.OpRemUint64: "RemU64",

	runtime.OpReturn: "Return",

	runtime.OpRightShift:  "RightShift",
	runtime.OpRightShiftU: "RightShiftU",

	runtime.OpSelect: "Select",

	runtime.OpField: "Field",

	runtime.OpSend: "Send",

	runtime.OpSetField: "SetField",

	runtime.OpSetMap: "SetMap",

	runtime.OpSetSlice: "SetSlice",

	runtime.OpSetVar: "SetVar",

	runtime.OpSlice: "Slice",

	runtime.OpStringSlice: "Slice",

	runtime.OpSubInt64:   "Sub",
	runtime.OpSubInt8:    "Sub8",
	runtime.OpSubInt16:   "Sub16",
	runtime.OpSubInt32:   "Sub32",
	runtime.OpSubFloat32: "Sub32",
	runtime.OpSubFloat64: "Sub",

	runtime.OpSubInvInt64:   "SubInv",
	runtime.OpSubInvInt8:    "SubInv8",
	runtime.OpSubInvInt16:   "SubInv16",
	runtime.OpSubInvInt32:   "SubInv32",
	runtime.OpSubInvFloat32: "SubInv32",
	runtime.OpSubInvFloat64: "SubInv",

	runtime.OpTailCall: "TailCall",

	runtime.OpTypify: "Typify",

	runtime.OpXor: "Xor",
}

var conditionName = [...]string{
	runtime.ConditionEqual:             "Equal",
	runtime.ConditionNotEqual:          "NotEqual",
	runtime.ConditionLess:              "Less",
	runtime.ConditionLessOrEqual:       "LessOrEqual",
	runtime.ConditionGreater:           "Greater",
	runtime.ConditionGreaterOrEqual:    "GreaterOrEqual",
	runtime.ConditionEqualLen:          "EqualLen",
	runtime.ConditionNotEqualLen:       "NotEqualLen",
	runtime.ConditionLessLen:           "LessLen",
	runtime.ConditionLessOrEqualLen:    "LessOrEqualLen",
	runtime.ConditionGreaterLen:        "GreaterOrEqualLen",
	runtime.ConditionGreaterOrEqualLen: "GreaterOrEqualLen",
	runtime.ConditionLessU:             "ConditionLessU",
	runtime.ConditionLessOrEqualU:      "ConditionLessOrEqualU",
	runtime.ConditionGreaterU:          "ConditionGreaterU",
	runtime.ConditionGreaterOrEqualU:   "ConditionGreaterOrEqualU",
	runtime.ConditionInterfaceNil:      "InterfaceNil",
	runtime.ConditionInterfaceNotNil:   "InterfaceNotNil",
	runtime.ConditionNil:               "Nil",
	runtime.ConditionNotNil:            "NotNil",
	runtime.ConditionOK:                "OK",
	runtime.ConditionNotOK:             "NotOK",
}
