// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/open2b/scriggo/internal/runtime"
)

type registerType int8

const (
	intRegister registerType = iota
	floatRegister
	stringRegister
	generalRegister
)

func (t registerType) String() string {
	switch t {
	case intRegister:
		return "int"
	case floatRegister:
		return "float"
	case stringRegister:
		return "string"
	case generalRegister:
		return "general"
	}
	panic("unknown type")
}

func packageName(pkg string) string {
	i := strings.LastIndex(pkg, "/")
	return pkg[i+1:]
}

// Disassemble disassembles the main function fn with the given globals,
// returning the assembly code for each package. The returned map has a
// package path as key and its assembly as value.
//
// n determines the maximum length, in runes, of the disassembled text in a
// Text instruction:
//
//   n > 0: at most n runes; leading and trailing white space are removed
//   n == 0: no text
//   n < 0: all text
//
func Disassemble(main *runtime.Function, globals []Global, n int) map[string][]byte {

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
		var line int
		if fn.Pos == nil {
			line = i + 1
		} else {
			line = fn.Pos.Line
		}
		if p, ok := functionsByPkg[fn.Pkg]; ok {
			p[fn] = line
		} else {
			functionsByPkg[fn.Pkg] = map[*runtime.Function]int{fn: line}
		}
		for _, sf := range fn.Functions {
			if sf.Name == "" {
				// Function literal.
				continue
			}
			if sf.Pkg != fn.Pkg {
				if packages, ok := importsByPkg[fn.Pkg]; ok {
					packages[sf.Pkg] = struct{}{}
				} else {
					importsByPkg[fn.Pkg] = map[string]struct{}{sf.Pkg: {}}
				}
			}
			added := false
			for _, f := range allFunctions {
				if f == sf {
					added = true
					break
				}
			}
			if !added {
				allFunctions = append(allFunctions, sf)
			}
		}
		for _, nf := range fn.NativeFunctions {
			if packages, ok := importsByPkg[fn.Pkg]; ok {
				packages[nf.Package()] = struct{}{}
			} else {
				importsByPkg[fn.Pkg] = map[string]struct{}{nf.Package(): {}}
			}
		}
	}

	assemblies := map[string][]byte{}

	var b bytes.Buffer

	for path, funcs := range functionsByPkg {

		b.WriteString("Package ")
		b.WriteString(packageName(path))
		b.WriteRune('\n')

		var packages []string
		if imports, ok := importsByPkg[path]; ok {
			for pkg := range imports {
				packages = append(packages, pkg)
			}
			sort.Slice(packages, func(i, j int) bool { return packages[i] < packages[j] })
			for _, pkg := range packages {
				b.WriteString("\nImport ")
				b.WriteString(strconv.Quote(pkg))
				b.WriteRune('\n')
			}
			packages = packages[:]
		}

		functions := make([]*runtime.Function, 0, len(funcs))
		for fn := range funcs {
			functions = append(functions, fn)
		}
		sort.Slice(functions, func(i, j int) bool { return funcs[functions[i]] < funcs[functions[j]] })

		for _, fn := range functions {
			if fn.Macro {
				b.WriteString("\nMacro ")
			} else {
				b.WriteString("\nFunc ")
			}
			b.WriteString(fn.Name)
			disassembleFunction(&b, globals, fn, n, 0)
		}

		assembly := make([]byte, b.Len())
		copy(assembly, b.Bytes())
		assemblies[path] = assembly

		b.Reset()

	}

	return assemblies
}

// DisassembleFunction disassembles the function fn with the given globals.
//
// n determines the maximum length, in runes, of the disassembled text in a
// Text instruction:
//
//   n > 0: at most n runes; leading and trailing white space are removed
//   n == 0: no text
//   n < 0: all text
//
func DisassembleFunction(fn *runtime.Function, globals []Global, n int) []byte {
	var b bytes.Buffer
	if fn.Macro {
		b.WriteString("Macro ")
	} else {
		b.WriteString("Func ")
	}
	b.WriteString(fn.Name)
	disassembleFunction(&b, globals, fn, n, 0)
	return b.Bytes()
}

func disassembleFunction(b *bytes.Buffer, globals []Global, fn *runtime.Function, textSize int, depth int) {
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
	if nIn := fn.Type.NumIn(); nIn > 0 || !fn.Macro {
		b.WriteByte('(')
		if nIn > 0 {
			out := map[registerType]int{intRegister: 0, floatRegister: 0, stringRegister: 0, generalRegister: 0}
			for i := 0; i < fn.Type.NumOut(); i++ {
				out[kindToType(fn.Type.Out(i).Kind())]++
			}
			in := map[registerType]int{intRegister: 0, floatRegister: 0, stringRegister: 0, generalRegister: 0}
			for i := 0; i < nIn; i++ {
				if i > 0 {
					b.WriteString(", ")
				}
				typ := fn.Type.In(i)
				label := registerKindToLabel(reflectToRegisterKind(typ.Kind()))
				vmType := kindToType(fn.Type.In(i).Kind())
				in[vmType]++
				reg := out[vmType] + in[vmType]
				_, _ = fmt.Fprintf(b, "%s%d %s", label, reg, typ)
			}
		}
		b.WriteByte(')')
	}

	// Print output parameters.
	if fn.Type.NumOut() > 0 {
		out := map[registerType]int{intRegister: 0, floatRegister: 0, stringRegister: 0, generalRegister: 0}
		b.WriteString(" (")
		for i := 0; i < fn.Type.NumOut(); i++ {
			if i > 0 {
				b.WriteString(", ")
			}
			typ := fn.Type.Out(i)
			label := registerKindToLabel(reflectToRegisterKind(typ.Kind()))
			vmType := kindToType(fn.Type.Out(i).Kind())
			out[vmType]++
			_, _ = fmt.Fprintf(b, "%s%d %s", label, out[vmType], fn.Type.Out(i))
		}
		b.WriteByte(')')
	}

	b.WriteByte('\n')
	_, _ = fmt.Fprintf(b, "%s\t; regs(%d,%d,%d,%d)\n", indent,
		fn.NumReg[intRegister], fn.NumReg[floatRegister], fn.NumReg[stringRegister], fn.NumReg[generalRegister])
	instrNum := runtime.Addr(len(fn.Body))
	for addr := runtime.Addr(0); addr < instrNum; addr++ {
		if label, ok := labelOf[runtime.Addr(addr)]; ok {
			_, _ = fmt.Fprintf(b, "%s%d:", indent, label)
		}
		in := fn.Body[addr]
		switch in.Op {
		case runtime.OpBreak, runtime.OpContinue, runtime.OpGoto:
			label := labelOf[runtime.Addr(decodeUint24(in.A, in.B, in.C))]
			_, _ = fmt.Fprintf(b, "%s\t%s %d", indent, operationName[in.Op], label)
		default:
			_, _ = fmt.Fprintf(b, "%s\t%s", indent, disassembleInstruction(fn, globals, addr, textSize))
		}
		// TODO: this part is not clear:
		if in.Op == runtime.OpLoadFunc && (int(in.B) < len(fn.Functions)) && fn.Functions[uint8(in.B)].Parent != nil { // function literal
			b.WriteByte(' ')
			b.WriteString(disassembleOperand(fn, in.C, reflect.Interface, false))
			b.WriteString(" func")
			disassembleFunction(b, globals, fn.Functions[uint8(in.B)], 0, depth+1)
		} else {
			b.WriteByte('\n')
		}
		switch in.Op {
		case runtime.OpCallFunc, runtime.OpCallMacro, runtime.OpCallIndirect, runtime.OpCallNative,
			runtime.OpTailCall, runtime.OpSlice, runtime.OpStringSlice:
			addr += 1
		case runtime.OpDefer:
			addr += 2
		}
		if in.Op == runtime.OpMakeSlice && in.B > 0 {
			addr += 1
		}
	}
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

func disassembleInstruction(fn *runtime.Function, globals []Global, addr runtime.Addr, textSize int) string {
	in := fn.Body[addr]
	op, a, b, c := in.Op, in.A, in.B, in.C
	k := false
	if op < 0 {
		op = -op
		k = true
	}
	s := operationName[op]
	switch op {
	case runtime.OpAdd, runtime.OpSub, runtime.OpSubInv, runtime.OpMul,
		runtime.OpDiv, runtime.OpRem, runtime.OpShl, runtime.OpShr:
		kind := reflect.Kind(a)
		s += " " + kind.String()
		s += " " + disassembleOperand(fn, b, kind, k)
		s += " " + disassembleOperand(fn, c, kind, false)
	case runtime.OpAddInt, runtime.OpSubInt, runtime.OpSubInvInt, runtime.OpMulInt,
		runtime.OpDivInt, runtime.OpRemInt, runtime.OpShlInt, runtime.OpShrInt:
		s += " " + disassembleOperand(fn, a, reflect.Int, false)
		s += " " + disassembleOperand(fn, b, reflect.Int, k)
		s += " " + disassembleOperand(fn, c, reflect.Int, false)
	case runtime.OpAddFloat64, runtime.OpSubFloat64, runtime.OpSubInvFloat64, runtime.OpMulFloat64,
		runtime.OpDivFloat64:
		s += " " + disassembleOperand(fn, a, reflect.Float64, false)
		s += " " + disassembleOperand(fn, b, reflect.Float64, k)
		s += " " + disassembleOperand(fn, c, reflect.Float64, false)
	case runtime.OpAnd, runtime.OpAndNot, runtime.OpOr, runtime.OpXor:
		s += " " + disassembleOperand(fn, a, reflect.Int, false)
		s += " " + disassembleOperand(fn, b, reflect.Int, k)
		s += " " + disassembleOperand(fn, c, reflect.Int, false)
	case runtime.OpAddr:
		s += " " + disassembleOperand(fn, a, reflect.Interface, false)
		s += " " + disassembleOperand(fn, b, reflect.Int, false)
		s += " " + disassembleOperand(fn, c, reflect.Interface, false)
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
	case runtime.OpCallFunc, runtime.OpCallMacro, runtime.OpCallIndirect, runtime.OpCallNative, runtime.OpTailCall, runtime.OpDefer:
		if a != runtime.CurrentFunction {
			switch op {
			case runtime.OpCallFunc, runtime.OpCallMacro, runtime.OpTailCall:
				sf := fn.Functions[uint8(a)]
				s += " " + packageName(sf.Pkg) + "." + sf.Name
			case runtime.OpCallIndirect:
				s += " " + "("
				s += disassembleOperand(fn, a, reflect.Interface, false)
				s += ")"
			case runtime.OpCallNative:
				nf := fn.NativeFunctions[uint8(a)]
				s += " " + packageName(nf.Package()) + "." + nf.Name()
			case runtime.OpDefer:
				s += " " + disassembleOperand(fn, a, reflect.Interface, false)
			}
		}
		grow := fn.Body[addr+1]
		stackShift := runtime.StackShift{int8(grow.Op), grow.A, grow.B, grow.C}
		if c != runtime.NoVariadicArgs && (op == runtime.OpCallIndirect || op == runtime.OpCallNative || op == runtime.OpDefer) {
			s += " ..." + strconv.Itoa(int(c))
		}
		_, _, typ := funcNameType(fn, a, addr, op)
		for i := 0; i < 4; i++ {
			s += " "
			if typ == nil || !funcHasParameterInRegister(typ, registerType(i)) {
				s += "_"
				continue
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
		case runtime.ConditionEqual, runtime.ConditionNotEqual,
			runtime.ConditionInterfaceEqual, runtime.ConditionInterfaceNotEqual,
			runtime.ConditionContainsElement, runtime.ConditionContainsKey,
			runtime.ConditionNotContainsElement, runtime.ConditionNotContainsKey:
			s += " " + disassembleOperand(fn, a, reflect.Interface, false)
			s += " " + conditionName[b]
			s += " " + disassembleOperand(fn, c, reflect.Interface, k)
		case runtime.ConditionContainsNil, runtime.ConditionNotContainsNil:
			s += " " + disassembleOperand(fn, a, reflect.Interface, false)
			s += " " + conditionName[b]
		default:
			s += " " + conditionName[b]
			s += " " + disassembleOperand(fn, a, reflect.Interface, false)
		}
	case runtime.OpIfInt:
		switch runtime.Condition(b) {
		case runtime.ConditionZero, runtime.ConditionNotZero:
			s += " " + conditionName[b]
			s += " " + disassembleOperand(fn, a, reflect.Int, false)
		case runtime.ConditionContainsRune, runtime.ConditionNotContainsRune:
			s += " " + disassembleOperand(fn, a, reflect.String, false)
			s += " " + conditionName[b]
			s += " " + disassembleOperand(fn, c, reflect.Int32, k)
		case runtime.ConditionContainsElement, runtime.ConditionContainsKey,
			runtime.ConditionNotContainsElement, runtime.ConditionNotContainsKey:
			s += " " + disassembleOperand(fn, a, reflect.Interface, false)
			s += " " + conditionName[b]
			s += " " + disassembleOperand(fn, c, reflect.Int, k)
		default:
			s += " " + disassembleOperand(fn, a, reflect.Int, false)
			s += " " + conditionName[b]
			s += " " + disassembleOperand(fn, c, reflect.Int, k)
		}
	case runtime.OpIfFloat:
		switch runtime.Condition(b) {
		case runtime.ConditionContainsElement, runtime.ConditionContainsKey,
			runtime.ConditionNotContainsElement, runtime.ConditionNotContainsKey:
			s += " " + disassembleOperand(fn, a, reflect.Interface, false)
			s += " " + conditionName[b]
			s += " " + disassembleOperand(fn, c, reflect.Float64, k)
		default:
			s += " " + disassembleOperand(fn, a, reflect.Float64, false)
			s += " " + conditionName[b]
			s += " " + disassembleOperand(fn, c, reflect.Float64, k)
		}
	case runtime.OpIfString:
		cond := runtime.Condition(b)
		if cond <= runtime.ConditionContainsSubstring {
			s += " " + disassembleOperand(fn, a, reflect.String, false)
			s += " " + conditionName[b]
			if cond <= runtime.ConditionLenEqual || cond == runtime.ConditionContainsSubstring {
				if k && c >= 0 {
					s += " " + strconv.Quote(string(byte(c)))
				} else {
					s += " " + disassembleOperand(fn, c, reflect.String, k)
				}
			} else {
				s += " " + disassembleOperand(fn, c, reflect.Int, k)
			}
		} else {
			s += " " + disassembleOperand(fn, a, reflect.Interface, false)
			s += " " + conditionName[b]
			s += " " + disassembleOperand(fn, a, reflect.String, k)
		}
	case runtime.OpField:
		s += " " + disassembleOperand(fn, a, reflect.Interface, false)
		s += " " + disassembleFieldIndex(fn.FieldIndexes[uint8(b)])
		s += " " + disassembleOperand(fn, c, getKind('c', fn, addr), false)
	case runtime.OpGetVar:
		s += " " + disassembleVarRef(fn, globals, int16(int(a)<<8|int(uint8(b))))
		s += " " + disassembleOperand(fn, c, getKind('c', fn, addr), false)
	case runtime.OpGetVarAddr:
		s += " " + disassembleVarRef(fn, globals, int16(int(a)<<8|int(uint8(b))))
		s += " " + disassembleOperand(fn, c, reflect.Interface, false)
	case runtime.OpGo, runtime.OpReturn:
	case runtime.OpIndex, runtime.OpIndexRef:
		s += " " + disassembleOperand(fn, a, reflect.Interface, false)
		s += " " + disassembleOperand(fn, b, reflect.Int, k)
		s += " " + disassembleOperand(fn, c, getKind('c', fn, addr), false)
	case runtime.OpIndexString:
		s += " " + disassembleOperand(fn, a, reflect.String, false)
		s += " " + disassembleOperand(fn, b, reflect.Int, k)
		s += " " + disassembleOperand(fn, c, reflect.Int, false)
	case runtime.OpLen:
		s += " " + strconv.Itoa(int(a))
		if registerType(a) == stringRegister {
			s += " " + disassembleOperand(fn, b, reflect.String, false)
		} else {
			s += " " + disassembleOperand(fn, b, reflect.Interface, false)
		}
		s += " " + disassembleOperand(fn, c, reflect.Int, false)
	case runtime.OpLoadFunc:
		if a == 0 {
			f := fn.Functions[uint8(b)]
			if f.Parent != nil { // f is a function literal.
				s = "Func" // overwrite s.
			} else {
				s += " " + packageName(f.Pkg) + "." + f.Name
				s += " " + disassembleOperand(fn, c, reflect.Interface, false)
			}
		} else { // LoadFunc (native).
			f := fn.NativeFunctions[uint8(b)]
			s += " " + packageName(f.Package()) + "." + f.Name()
			s += " " + disassembleOperand(fn, c, reflect.Interface, false)
		}
	case runtime.OpLoad:
		t, i := decodeValueIndex(a, b)
		switch t {
		case intRegister:
			s += " " + fmt.Sprintf("%d", fn.Values.Int[i])
			s += " " + disassembleOperand(fn, c, reflect.Int, false)
		case floatRegister:
			s += " " + fmt.Sprintf("%f", fn.Values.Float[i])
			s += " " + disassembleOperand(fn, c, reflect.Float64, false)
		case stringRegister:
			s += " " + fmt.Sprintf("%q", fn.Values.String[i])
			s += " " + disassembleOperand(fn, c, reflect.String, false)
		case generalRegister:
			v := fn.Values.General[i]
			if v.IsValid() {
				s += " " + fmt.Sprintf("%#v", v.Interface())
			} else {
				s += " nil"
			}
			s += " " + disassembleOperand(fn, c, reflect.Interface, false)
		}
	case runtime.OpMakeArray, runtime.OpMakeStruct, runtime.OpNew:
		s += " " + fn.Types[int(uint(b))].String()
		s += " " + disassembleOperand(fn, c, reflect.Interface, false)
	case runtime.OpMakeChan, runtime.OpMakeMap:
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
		switch registerType(a) {
		case intRegister:
			s += " " + disassembleOperand(fn, b, reflect.Int, k)
			s += " " + disassembleOperand(fn, c, reflect.Int, false)
		case floatRegister:
			s += " " + disassembleOperand(fn, b, reflect.Float64, k)
			s += " " + disassembleOperand(fn, c, reflect.Float64, false)
		case stringRegister:
			s += " " + disassembleOperand(fn, b, reflect.String, k)
			s += " " + disassembleOperand(fn, c, reflect.String, false)
		case generalRegister:
			s += " " + disassembleOperand(fn, b, reflect.Interface, k)
			s += " " + disassembleOperand(fn, c, reflect.Interface, false)
		}
	case runtime.OpNeg:
		kind := reflect.Kind(a)
		if kind != reflect.Int && kind != reflect.Float64 {
			s += " " + kind.String()
		}
		s += " " + disassembleOperand(fn, b, kind, false)
		s += " " + disassembleOperand(fn, c, kind, false)
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
	case runtime.OpSelect:
		// No operand.
	case runtime.OpSetField:
		s += " " + disassembleOperand(fn, a, getKind('a', fn, addr), k)
		s += " " + disassembleOperand(fn, b, reflect.Interface, false)
		s += " " + disassembleFieldIndex(fn.FieldIndexes[uint8(c)])
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
		s += " " + disassembleOperand(fn, a, getKind('a', fn, addr), k)
		s += " " + disassembleVarRef(fn, globals, int16(int(b)<<8|int(uint8(c))))
	case runtime.OpShow:
		typ := fn.Types[int(uint(a))]
		s += " " + typ.String()
		s += " " + disassembleOperand(fn, b, reflectToRegisterKind(typ.Kind()), false)
		ctx, _, _ := decodeRenderContext(runtime.Context(c))
		s += " " + "(" + ctx.String() + ")"
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
	case runtime.OpText:
		if textSize != 0 {
			i := int(decodeUint16(a, b))
			s += " " + disassembleText(fn.Text[i], textSize)
		}
	case runtime.OpTypify:
		typ := fn.Types[int(uint(a))]
		s += " " + typ.String()
		s += " " + disassembleOperand(fn, b, reflectToRegisterKind(typ.Kind()), k)
		s += " " + disassembleOperand(fn, c, reflect.Interface, false)
	case runtime.OpZero:
		if a >= 10 {
			a -= 10
			s = "NotZero"
		}
		var kind reflect.Kind
		switch registerType(a) {
		case intRegister:
			kind = reflect.Int
		case floatRegister:
			kind = reflect.Float64
		case stringRegister:
			kind = reflect.String
		case generalRegister:
			kind = reflect.Interface
		}
		s += " " + disassembleOperand(fn, b, kind, false)
		s += " " + disassembleOperand(fn, c, reflect.Bool, false)
	default:
		panic(fmt.Sprintf("unknown operation %d", op))
	}
	return s
}

// funcNameType returns a boolean indications if the specific function is a
// macro, its name and its type. If the function is not available. Only one of
// index and addr is meaningful, depending on the operation specified by op.
func funcNameType(fn *runtime.Function, index int8, addr runtime.Addr, op runtime.Operation) (bool, string, reflect.Type) {
	switch op {
	case runtime.OpCallFunc, runtime.OpCallMacro:
		macro := fn.Functions[index].Macro
		typ := fn.Functions[index].Type
		name := fn.Functions[index].Name
		return macro, name, typ
	case runtime.OpCallNative:
		name := fn.NativeFunctions[index].Name()
		typ := reflect.TypeOf(fn.NativeFunctions[index].Func())
		return false, name, typ
	case runtime.OpCallIndirect, runtime.OpDefer:
		return false, "", fn.DebugInfo[addr].FuncType
	case runtime.OpTailCall:
		panic("BUG: not implemented") // TODO.
	default:
		panic("BUG")
	}
}

// funcHasParameterInRegister reports whether the given function has at least
// one parameter (input or output) that is stored into the given register type.
func funcHasParameterInRegister(fn reflect.Type, reg registerType) bool {
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
	macro, name, typ := funcNameType(fn, index, addr, op)
	if typ == nil {
		return ""
	}
	if name == "$initvars" {
		return "package vars init"
	}
	print := func(t reflect.Type) string {
		str := ""
		switch kindToType(t.Kind()) {
		case intRegister:
			stackShift[0]++
			str += fmt.Sprintf("i%d %v", stackShift[0], t)
		case floatRegister:
			stackShift[1]++
			str += fmt.Sprintf("f%d %v", stackShift[1], t)
		case stringRegister:
			stackShift[2]++
			str += fmt.Sprintf("s%d %v", stackShift[2], t)
		case generalRegister:
			stackShift[3]++
			str += fmt.Sprintf("g%d %v", stackShift[3], t)
		}
		return str
	}
	var sout string
	if nout := typ.NumOut(); nout > 0 {
		sout += " ("
		for i := 0; i < nout; i++ {
			if i > 0 {
				sout += ", "
			}
			sout += print(typ.Out(i))
		}
		sout += ")"
	}
	s := "func("
	if macro {
		if typ.NumIn() == 0 {
			return "macro"
		}
		s = "macro("
	}
	lastIn := typ.NumIn() - 1
	for i := 0; i < lastIn; i++ {
		s += print(typ.In(i)) + ", "
	}
	if lastIn >= 0 {
		if variadic == runtime.NoVariadicArgs || variadic == 0 {
			s += print(typ.In(lastIn))
		} else {
			varType := typ.In(lastIn).Elem()
			for i := int8(0); i < variadic; i++ {
				s += print(varType)
				if i < variadic-1 {
					s += ", "
				}
			}
		}
	}
	s += ")" + sout
	return s
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

func disassembleText(txt []byte, size int) string {
	if size < 0 {
		size = maxInt
	}
	if len(txt) <= size {
		return strconv.Quote(string(txt))
	}
	var b strings.Builder
	if r, _ := utf8.DecodeRune(txt); unicode.IsSpace(r) {
		b.WriteRune('…')
		if size == 1 {
			return strconv.Quote(b.String())
		}
		size--
	}
	r, _ := utf8.DecodeLastRune(txt)
	truncated := unicode.IsSpace(r)
	txt = bytes.TrimSpace(txt)
	if len(txt) > size {
		size--
		var p int
		for i := 0; i < size; i++ {
			_, s := utf8.DecodeRune(txt)
			p += s
		}
		txt = txt[:p]
		truncated = true
	}
	b.Write(txt)
	if truncated {
		b.WriteRune('…')
	}
	return strconv.Quote(b.String())
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
		case reflect.Int <= kind && kind <= reflect.Int64:
			return strconv.Itoa(int(op))
		case reflect.Uint <= kind && kind <= reflect.Uintptr:
			return strconv.Itoa(int(uint8(op)))
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
			return strconv.Quote(fn.Values.String[uint8(op)])
		case kind == reflect.Invalid:
			return "?"
		default:
			v := fn.Values.General[uint8(op)]
			if v.IsValid() {
				return fmt.Sprintf("%#v", v.Interface())
			}
			return "nil"
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

func disassembleFieldIndex(index []int) string {
	if len(index) == 1 {
		return strconv.Itoa(index[0])
	}
	var s strings.Builder
	for i, v := range index {
		if i > 0 {
			s.WriteString(",")
		}
		s.WriteString(strconv.Itoa(v))
	}
	return s.String()
}

var operationName = [...]string{

	runtime.OpNone: "Nop",

	runtime.OpAdd:        "Add",
	runtime.OpAddInt:     "Add",
	runtime.OpAddFloat64: "Add",

	runtime.OpAddr: "Addr",

	runtime.OpAnd: "And",

	runtime.OpAndNot: "AndNot",

	runtime.OpAppend: "Append",

	runtime.OpAppendSlice: "AppendSlice",

	runtime.OpAssert: "Assert",

	runtime.OpBreak: "Break",

	runtime.OpCallFunc:     "Call",
	runtime.OpCallIndirect: "Call",
	runtime.OpCallMacro:    "Call",
	runtime.OpCallNative:   "Call",

	runtime.OpCap: "Cap",

	runtime.OpCase: "Case",

	runtime.OpClose: "Close",

	runtime.OpComplex64:  "Complex64",
	runtime.OpComplex128: "Complex128",

	runtime.OpConcat: "Concat",

	runtime.OpContinue: "Continue",

	runtime.OpConvert:       "Convert",
	runtime.OpConvertInt:    "Convert",
	runtime.OpConvertUint:   "ConvertU",
	runtime.OpConvertFloat:  "Convert",
	runtime.OpConvertString: "Convert",

	runtime.OpCopy: "Copy",

	runtime.OpDefer: "Defer",

	runtime.OpDelete: "Delete",

	runtime.OpDiv:        "Div",
	runtime.OpDivInt:     "Div",
	runtime.OpDivFloat64: "Div",

	runtime.OpField: "Field",

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

	runtime.OpIndexRef: "IndexRef",

	runtime.OpLen: "Len",

	runtime.OpLoad: "Load",

	runtime.OpLoadFunc: "LoadFunc",

	runtime.OpMakeArray: "MakeArray",

	runtime.OpMakeChan: "MakeChan",

	runtime.OpMakeMap: "MakeMap",

	runtime.OpMakeSlice: "MakeSlice",

	runtime.OpMakeStruct: "MakeStruct",

	runtime.OpMapIndex: "MapIndex",

	runtime.OpMethodValue: "MethodValue",

	runtime.OpMove: "Move",

	runtime.OpMul:        "Mul",
	runtime.OpMulInt:     "Mul",
	runtime.OpMulFloat64: "Mul",

	runtime.OpNeg: "Neg",

	runtime.OpNew: "New",

	runtime.OpOr: "Or",

	runtime.OpPanic: "Panic",

	runtime.OpPrint: "Print",

	runtime.OpRange: "Range",

	runtime.OpRangeString: "Range",

	runtime.OpRealImag: "RealImag",

	runtime.OpReceive: "Receive",

	runtime.OpRecover: "Recover",

	runtime.OpRem:    "Rem",
	runtime.OpRemInt: "Rem",

	runtime.OpReturn: "Return",

	runtime.OpSelect: "Select",

	runtime.OpSend: "Send",

	runtime.OpSetField: "SetField",

	runtime.OpSetMap: "SetMap",

	runtime.OpSetSlice: "SetSlice",

	runtime.OpSetVar: "SetVar",

	runtime.OpShl:    "Shl",
	runtime.OpShlInt: "Shl",

	runtime.OpShow: "Show",

	runtime.OpShr:    "Shr",
	runtime.OpShrInt: "Shr",

	runtime.OpSlice: "Slice",

	runtime.OpStringSlice: "Slice",

	runtime.OpSub:        "Sub",
	runtime.OpSubInt:     "Sub",
	runtime.OpSubFloat64: "Sub",

	runtime.OpSubInv:        "SubInv",
	runtime.OpSubInvInt:     "SubInv",
	runtime.OpSubInvFloat64: "SubInv",

	runtime.OpTailCall: "TailCall",

	runtime.OpText: "Text",

	runtime.OpTypify: "Typify",

	runtime.OpXor: "Xor",

	runtime.OpZero: "Zero",
}

var conditionName = [...]string{
	runtime.ConditionZero:                 "Zero",
	runtime.ConditionNotZero:              "NotZero",
	runtime.ConditionEqual:                "Equal",
	runtime.ConditionNotEqual:             "NotEqual",
	runtime.ConditionLess:                 "Less",
	runtime.ConditionLessEqual:            "LessEqual",
	runtime.ConditionGreater:              "Greater",
	runtime.ConditionGreaterEqual:         "GreaterEqual",
	runtime.ConditionLessU:                "Less",
	runtime.ConditionLessEqualU:           "LessEqual",
	runtime.ConditionGreaterU:             "Greater",
	runtime.ConditionGreaterEqualU:        "GreaterEqual",
	runtime.ConditionLenEqual:             "LenEqual",
	runtime.ConditionLenNotEqual:          "LenNotEqual",
	runtime.ConditionLenLess:              "LenLess",
	runtime.ConditionLenLessEqual:         "LenLessEqual",
	runtime.ConditionLenGreater:           "LenGreater",
	runtime.ConditionLenGreaterEqual:      "LenGreaterEqual",
	runtime.ConditionInterfaceNil:         "InterfaceNil",
	runtime.ConditionInterfaceNotNil:      "InterfaceNotNil",
	runtime.ConditionNil:                  "Nil",
	runtime.ConditionNotNil:               "NotNil",
	runtime.ConditionContainsSubstring:    "ContainsSubstring",
	runtime.ConditionContainsRune:         "ContainsRune",
	runtime.ConditionContainsElement:      "ContainsElement",
	runtime.ConditionContainsKey:          "ContainsKey",
	runtime.ConditionContainsNil:          "ContainsNil",
	runtime.ConditionNotContainsSubstring: "NotContainsSubstring",
	runtime.ConditionNotContainsRune:      "NotContainsRune",
	runtime.ConditionNotContainsElement:   "NotContainsElement",
	runtime.ConditionNotContainsKey:       "NotContainsKey",
	runtime.ConditionNotContainsNil:       "NotContainsNil",
	runtime.ConditionOK:                   "OK",
	runtime.ConditionNotOK:                "NotOK",
}
