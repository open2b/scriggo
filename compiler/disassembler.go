// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"scrigo/vm"
)

func packageName(pkg string) string {
	i := strings.LastIndex(pkg, "/")
	return pkg[i+1:]
}

func DisassembleDir(dir string, main *vm.ScrigoFunction) (err error) {

	packages, err := Disassemble(main)
	if err != nil {
		return err
	}

	var fi *os.File
	defer func() {
		if fi != nil {
			err2 := fi.Close()
			if err == nil {
				err = err2
			}
		}
	}()

	for path, source := range packages {
		pkgDir, file := filepath.Split(path)
		fullDir := filepath.Join(dir, pkgDir)
		err = os.MkdirAll(fullDir, 0775)
		if err != nil {
			return err
		}
		fi, err = os.OpenFile(filepath.Join(fullDir, file+".s"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
		if err != nil {
			return err
		}
		_, err = fi.WriteString(source)
		if err != nil {
			return err
		}
		err = fi.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func Disassemble(main *vm.ScrigoFunction) (assembler map[string]string, err error) {

	functionsByPkg := map[string]map[*vm.ScrigoFunction]int{}
	importsByPkg := map[string]map[string]struct{}{}

	c := len(main.ScrigoFunctions)
	if c == 0 {
		c = 1
	}
	allFunctions := make([]*vm.ScrigoFunction, 1, c)
	allFunctions[0] = main

	for i := 0; i < len(allFunctions); i++ {
		fn := allFunctions[i]
		if p, ok := functionsByPkg[fn.Pkg]; ok {
			p[fn] = fn.Line
		} else {
			functionsByPkg[fn.Pkg] = map[*vm.ScrigoFunction]int{fn: fn.Line}
		}
		for _, sf := range fn.ScrigoFunctions {
			if sf.Pkg != fn.Pkg {
				if packages, ok := importsByPkg[fn.Pkg]; ok {
					packages[sf.Pkg] = struct{}{}
				} else {
					importsByPkg[fn.Pkg] = map[string]struct{}{sf.Pkg: {}}
				}
			}
		}
		for _, nf := range fn.NativeFunctions {
			if packages, ok := importsByPkg[fn.Pkg]; ok {
				packages[nf.Pkg] = struct{}{}
			} else {
				importsByPkg[fn.Pkg] = map[string]struct{}{nf.Pkg: {}}
			}
		}
		allFunctions = append(allFunctions, fn.ScrigoFunctions...)
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

		functions := make([]*vm.ScrigoFunction, 0, len(funcs))
		for fn := range funcs {
			functions = append(functions, fn)
		}
		sort.Slice(functions, func(i, j int) bool { return functions[i].Line < functions[i].Line })

		for _, fn := range functions {
			_, _ = b.WriteString("\nFunc ")
			_, _ = b.WriteString(fn.Name)
			disassembleFunction(&b, fn, 0)
		}
		_, _ = b.WriteRune('\n')

		assembler[path] = b.String()

		b.Reset()

	}

	return assembler, nil
}

func DisassembleFunction(w io.Writer, fn *vm.ScrigoFunction) (int64, error) {
	var b bytes.Buffer
	_, _ = fmt.Fprintf(&b, "Func %s", fn.Name)
	disassembleFunction(&b, fn, 0)
	return b.WriteTo(w)
}

func disassembleFunction(w *bytes.Buffer, fn *vm.ScrigoFunction, depth int) {
	indent := ""
	if depth > 0 {
		indent = strings.Repeat("\t", depth)
	}
	labelOf := map[uint32]uint32{}
	for _, in := range fn.Body {
		switch in.Op {
		case vm.OpBreak, vm.OpContinue, vm.OpGoto:
			labelOf[decodeAddr(in.A, in.B, in.C)] = 0
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
			labelOf[uint32(addr)] = uint32(i) + 1
		}
	}
	_, _ = fmt.Fprintf(w, "(")
	nIn := fn.Type.NumIn()
	nOut := fn.Type.NumOut()
	for i := 0; i < nIn; i++ {
		if i > 0 {
			_, _ = fmt.Fprint(w, ", ")
		}
		typ := fn.Type.In(i)
		label := registerKindToLabel(reflectToRegisterKind(typ.Kind()))
		_, _ = fmt.Fprintf(w, "%s%d %s", label, nOut+i+1, typ)
	}
	_, _ = fmt.Fprint(w, ")")
	if nOut > 0 {
		_, _ = fmt.Fprint(w, " (")
		for i := 0; i < nOut; i++ {
			if i > 0 {
				_, _ = fmt.Fprint(w, ", ")
			}
			typ := fn.Type.Out(i)
			label := registerKindToLabel(reflectToRegisterKind(typ.Kind()))
			_, _ = fmt.Fprintf(w, "%s%d %s", label, i+1, fn.Type.Out(i))
		}
		_, _ = fmt.Fprint(w, ")")
	}
	_, _ = fmt.Fprint(w, "\n")
	_, _ = fmt.Fprintf(w, "%s\t// regs(%d,%d,%d,%d)\n", indent,
		fn.RegNum[0], fn.RegNum[1], fn.RegNum[2], fn.RegNum[3])
	instrNum := uint32(len(fn.Body))
	for addr := uint32(0); addr < instrNum; addr++ {
		if label, ok := labelOf[uint32(addr)]; ok {
			_, _ = fmt.Fprintf(w, "%s%d:", indent, label)
		}
		in := fn.Body[addr]
		switch in.Op {
		case vm.OpBreak, vm.OpContinue, vm.OpGoto:
			label := labelOf[decodeAddr(in.A, in.B, in.C)]
			_, _ = fmt.Fprintf(w, "%s\t%s %d\n", indent, operationName[in.Op], label)
		case vm.OpFunc:
			_, _ = fmt.Fprintf(w, "%s\tFunc %s ", indent, disassembleOperand(fn, in.C, vm.Interface, false))
			disassembleFunction(w, fn.Literals[uint8(in.B)], depth+1)
		default:
			_, _ = fmt.Fprintf(w, "%s\t%s\n", indent, disassembleInstruction(fn, addr))
		}
		switch in.Op {
		case vm.OpCall, vm.OpCallIndirect, vm.OpCallNative, vm.OpTailCall, vm.OpMakeSlice:
			addr += 1
		case vm.OpDefer:
			addr += 2
		}
	}
}

func DisassembleInstruction(w io.Writer, fn *vm.ScrigoFunction, addr uint32) (int64, error) {
	n, err := io.WriteString(w, disassembleInstruction(fn, addr))
	return int64(n), err
}

func disassembleInstruction(fn *vm.ScrigoFunction, addr uint32) string {
	in := fn.Body[addr]
	op, a, b, c := in.Op, in.A, in.B, in.C
	k := false
	if op < 0 {
		op = -op
		k = true
	}
	s := operationName[op]
	switch op {
	case vm.OpAddInt, vm.OpAddInt8, vm.OpAddInt16, vm.OpAddInt32,
		vm.OpAnd, vm.OpAndNot, vm.OpOr, vm.OpXor,
		vm.OpDivInt, vm.OpDivInt8, vm.OpDivInt16, vm.OpDivInt32, vm.OpDivUint8, vm.OpDivUint16, vm.OpDivUint32, vm.OpDivUint64,
		vm.OpMulInt, vm.OpMulInt8, vm.OpMulInt16, vm.OpMulInt32,
		vm.OpRemInt, vm.OpRemInt8, vm.OpRemInt16, vm.OpRemInt32, vm.OpRemUint8, vm.OpRemUint16, vm.OpRemUint32, vm.OpRemUint64,
		vm.OpSubInt, vm.OpSubInt8, vm.OpSubInt16, vm.OpSubInt32,
		vm.OpSubInvInt, vm.OpSubInvInt8, vm.OpSubInvInt16, vm.OpSubInvInt32,
		vm.OpLeftShift, vm.OpLeftShift8, vm.OpLeftShift16, vm.OpLeftShift32,
		vm.OpRightShift, vm.OpRightShiftU:
		s += " " + disassembleOperand(fn, a, vm.Int, false)
		s += " " + disassembleOperand(fn, b, vm.Int, k)
		s += " " + disassembleOperand(fn, c, vm.Int, false)
	case vm.OpAddFloat32, vm.OpAddFloat64, vm.OpDivFloat32, vm.OpDivFloat64,
		vm.OpMulFloat32, vm.OpMulFloat64,
		vm.OpSubFloat32, vm.OpSubFloat64, vm.OpSubInvFloat32, vm.OpSubInvFloat64:
		s += " " + disassembleOperand(fn, a, vm.Float64, false)
		s += " " + disassembleOperand(fn, b, vm.Float64, k)
		s += " " + disassembleOperand(fn, c, vm.Float64, false)
	case vm.OpAssert:
		s += " " + disassembleOperand(fn, a, vm.Interface, false)
		s += " type(" + fn.Types[b].String() + ")"
		t := fn.Types[int(uint(b))]
		var kind = reflectToRegisterKind(t.Kind())
		s += " " + disassembleOperand(fn, c, kind, false)
	case vm.OpBind, vm.OpGetVar:
		s += " " + disassembleVarRef(fn, int16(int(a)<<8|int(uint8(b))))
		s += " " + disassembleOperand(fn, c, vm.Int, false)
	case vm.OpBreak, vm.OpContinue, vm.OpGoto:
		s += " " + strconv.Itoa(int(decodeAddr(a, b, c)))
	case vm.OpCall, vm.OpCallIndirect, vm.OpCallNative, vm.OpTailCall, vm.OpDefer:
		if a != vm.CurrentFunction {
			switch op {
			case vm.OpCall, vm.OpTailCall:
				sf := fn.ScrigoFunctions[uint8(a)]
				s += " " + packageName(sf.Pkg) + "." + sf.Name
			case vm.OpCallNative:
				nf := fn.NativeFunctions[uint8(a)]
				s += " " + packageName(nf.Pkg) + "." + nf.Name
			case vm.OpCallIndirect, vm.OpDefer:
				s += " " + disassembleOperand(fn, a, vm.Interface, false)
			}
		}
		if c != vm.NoVariadic && (op == vm.OpCallIndirect || op == vm.OpCallNative || op == vm.OpDefer) {
			s += " ..." + strconv.Itoa(int(c))
		}
		switch op {
		case vm.OpCallIndirect, vm.OpDefer:
			grow := fn.Body[addr+1]
			s += "\t// Stack shift: " + strconv.Itoa(int(grow.Op)) + ", " + strconv.Itoa(int(grow.A)) + ", " +
				strconv.Itoa(int(grow.B)) + ", " + strconv.Itoa(int(grow.C))
		case vm.OpCall, vm.OpCallNative:
			grow := fn.Body[addr+1]
			stackShift := vm.StackShift{int8(grow.Op), grow.A, grow.B, grow.C}
			s += "\t// " + disassembleFunctionCall(fn, a, op == vm.OpCallNative, stackShift, c)
		}
		if op == vm.OpDefer {
			args := fn.Body[addr+2]
			s += "; args: " + strconv.Itoa(int(args.Op)) + ", " + strconv.Itoa(int(args.A)) + ", " +
				strconv.Itoa(int(args.B)) + ", " + strconv.Itoa(int(args.C))
		}
	case vm.OpCap:
		s += " " + disassembleOperand(fn, a, vm.Interface, false)
		s += " " + disassembleOperand(fn, c, vm.Int, false)
	case vm.OpCase:
		switch reflect.SelectDir(a) {
		case reflect.SelectSend:
			s += " send " + disassembleOperand(fn, b, vm.Int, k) + " " + disassembleOperand(fn, c, vm.Interface, false)
		case reflect.SelectRecv:
			s += " recv " + disassembleOperand(fn, b, vm.Int, false) + " " + disassembleOperand(fn, c, vm.Interface, false)
		default:
			s += " default"
		}
	case vm.OpClose, vm.OpPanic, vm.OpPrint:
		s += " " + disassembleOperand(fn, a, vm.Interface, false)
	case vm.OpCopy:
		s += " " + disassembleOperand(fn, a, vm.Interface, false)
		s += " " + disassembleOperand(fn, b, vm.Interface, false)
		s += " " + disassembleOperand(fn, c, vm.Int, false)
	case vm.OpConcat:
		s += " " + disassembleOperand(fn, a, vm.String, false)
		s += " " + disassembleOperand(fn, b, vm.String, k)
		s += " " + disassembleOperand(fn, c, vm.String, false)
	case vm.OpConvert:
		s += " " + disassembleOperand(fn, a, vm.Interface, false)
		s += " " + fn.Types[int(uint(b))].String()
		s += " " + disassembleOperand(fn, c, vm.Interface, false)
	case vm.OpConvertInt, vm.OpConvertUint:
		s += " " + disassembleOperand(fn, a, vm.Int, false)
		s += " " + fn.Types[int(uint(b))].String()
		s += " " + disassembleOperand(fn, c, vm.Interface, false)
	case vm.OpConvertFloat:
		s += " " + disassembleOperand(fn, a, vm.Float64, false)
		s += " " + fn.Types[int(uint(b))].String()
		s += " " + disassembleOperand(fn, c, vm.Interface, false)
	case vm.OpConvertString:
		s += " " + disassembleOperand(fn, a, vm.String, false)
		s += " " + fn.Types[int(uint(b))].String()
		s += " " + disassembleOperand(fn, c, vm.Interface, false)
	case vm.OpDelete:
		s += " " + disassembleOperand(fn, a, vm.Interface, false)
		s += " " + disassembleOperand(fn, b, vm.Interface, false)
	case vm.OpIf:
		if vm.Condition(b) == vm.ConditionOK {
			s += " OK"
		} else {
			s += " " + disassembleOperand(fn, a, vm.Interface, false)
			s += " " + conditionName[b]
		}
	case vm.OpIfInt, vm.OpIfUint:
		s += " " + disassembleOperand(fn, a, vm.Int, false)
		s += " " + conditionName[b]
		if vm.Condition(b) >= vm.ConditionEqual {
			s += " " + disassembleOperand(fn, c, vm.Int, k)
		}
	case vm.OpIfFloat:
		s += " " + disassembleOperand(fn, a, vm.Float64, false)
		s += " " + conditionName[b]
		s += " " + disassembleOperand(fn, c, vm.Float64, k)
	case vm.OpIfString:
		s += " " + disassembleOperand(fn, a, vm.String, false)
		s += " " + conditionName[b]
		if vm.Condition(b) < vm.ConditionEqualLen {
			if k && c >= 0 {
				s += " " + strconv.Quote(string(c))
			} else {
				s += " " + disassembleOperand(fn, c, vm.String, k)
			}
		} else {
			s += " " + disassembleOperand(fn, c, vm.Int, k)
		}
	case vm.OpFunc:
		s += " func(" + strconv.Itoa(int(uint8(b))) + ")"
		s += " " + disassembleOperand(fn, c, vm.Int, false)
	case vm.OpGetFunc:
		if a == 0 {
			f := fn.ScrigoFunctions[uint8(b)]
			s += " " + packageName(f.Pkg) + "." + f.Name
		} else {
			f := fn.NativeFunctions[uint8(b)]
			s += " " + packageName(f.Pkg) + "." + f.Name
		}
		s += " " + disassembleOperand(fn, c, vm.Interface, false)
	case vm.OpGo, vm.OpReturn:
	case vm.OpIndex:
		s += " " + disassembleOperand(fn, a, vm.Interface, false)
		s += " " + disassembleOperand(fn, b, vm.Int, k)
		s += " " + disassembleOperand(fn, c, vm.Interface, false)
	case vm.OpLen:
		s += " " + strconv.Itoa(int(a))
		if a == 0 {
			s += " " + disassembleOperand(fn, b, vm.String, false)
		} else {
			s += " " + disassembleOperand(fn, b, vm.Interface, false)
		}
		s += " " + disassembleOperand(fn, c, vm.Int, false)
	case vm.OpLoadNumber:
		if a == 0 {
			s += " int"
			s += " " + fmt.Sprintf("%d", fn.Constants.Int[uint8(b)])
			s += " " + disassembleOperand(fn, c, vm.Int, false)
		} else {
			s += " float"
			s += " " + fmt.Sprintf("%f", fn.Constants.Float[uint8(b)])
			s += " " + disassembleOperand(fn, c, vm.Float64, false)
		}
	case vm.OpMakeChan:
		s += " " + fn.Types[int(uint(a))].String()
		s += " " + disassembleOperand(fn, b, vm.Int, k)
		s += " " + disassembleOperand(fn, c, vm.Interface, false)
	case vm.OpMakeMap:
		s += " " + fn.Types[int(uint(a))].String()
		s += " " + disassembleOperand(fn, b, vm.Int, k)
		s += " " + disassembleOperand(fn, c, vm.Interface, false)
	case vm.OpMapIndex:
		//s += " " + disassembleOperand(scrigo, a, vm.Interface, false)
		//key := reflectToRegisterKind()
		//s += " " + disassembleOperand(scrigo, b, vm.Interface, false)
		//s += " " + disassembleOperand(scrigo, c, vm.Interface, false)
	case vm.OpMove:
		switch vm.Type(a) {
		case vm.TypeInt:
			s += " " + disassembleOperand(fn, b, vm.Int, k)
			s += " " + disassembleOperand(fn, c, vm.Int, false)
		case vm.TypeFloat:
			s += " " + disassembleOperand(fn, b, vm.Float64, k)
			s += " " + disassembleOperand(fn, c, vm.Float64, false)
		case vm.TypeString:
			s += " " + disassembleOperand(fn, b, vm.String, k)
			s += " " + disassembleOperand(fn, c, vm.String, false)
		case vm.TypeGeneral:
			s += " " + disassembleOperand(fn, b, vm.Interface, k)
			s += " " + disassembleOperand(fn, c, vm.Interface, false)
		}
	case vm.OpNew:
		s += " " + fn.Types[int(uint(b))].String()
		s += " " + disassembleOperand(fn, c, vm.Interface, false)
	case vm.OpRange:
		s += " " + disassembleOperand(fn, a, vm.Interface, false)
		s += " " + disassembleOperand(fn, b, vm.Int, false)
		s += " " + disassembleOperand(fn, c, vm.Int, false)
	case vm.OpRangeString:
		s += " " + disassembleOperand(fn, a, vm.String, k)
		s += " " + disassembleOperand(fn, b, vm.Int, false)
		s += " " + disassembleOperand(fn, c, vm.Int, false)
	case vm.OpReceive:
		s += " " + disassembleOperand(fn, a, vm.Interface, false)
		s += " " + disassembleOperand(fn, b, vm.Bool, false)
		s += " " + disassembleOperand(fn, c, vm.Interface, false)
	case vm.OpRecover:
		if c != 0 {
			s += " " + disassembleOperand(fn, c, vm.Interface, false)
		}
	case vm.OpSelector:
		//s += " " + disassembleOperand(scrigo, c, vm.Interface, false)
	case vm.OpSend:
		s += " " + disassembleOperand(fn, a, vm.Interface, false)
		s += " " + disassembleOperand(fn, c, vm.Interface, false)
	case vm.OpMakeSlice:
		s += " " + fn.Types[int(uint(a))].String()
		s += " " + disassembleOperand(fn, c, vm.Interface, false)
		s += " // len: "
		s += fmt.Sprintf("%d", fn.Body[addr+1].A)
		s += ", cap: "
		s += fmt.Sprintf("%d", fn.Body[addr+1].B)
	case vm.OpSetMap:
		s += " " + disassembleOperand(fn, a, vm.Interface, false)
		if k {
			s += fmt.Sprintf(" K(%v)", b)
		} else {
			s += " " + disassembleOperand(fn, b, vm.Interface, false)
		}
		s += " " + disassembleOperand(fn, c, vm.Interface, false)
	case vm.OpSetSlice:
		s += " " + disassembleOperand(fn, a, vm.Interface, false)
		s += " " + disassembleOperand(fn, b, vm.Int, k)
		s += " " + disassembleOperand(fn, c, vm.Int, false)
	case vm.OpSetVar:
		s += " " + disassembleOperand(fn, a, vm.Int, op < 0)
		s += " " + disassembleVarRef(fn, int16(int(b)<<8|int(uint8(c))))
	case vm.OpSliceIndex:
		s += " " + disassembleOperand(fn, a, vm.Interface, false)
		s += " " + disassembleOperand(fn, b, vm.Int, k)
		//s += " " + disassembleOperand(scrigo, c, vm.Interface, false)
	}
	return s
}

func disassembleFunctionCall(fn *vm.ScrigoFunction, index int8, isNative bool, stackShift vm.StackShift, variadic int8) string {
	var funcType reflect.Type
	var funcName string
	if isNative {
		funcType = reflect.TypeOf(fn.NativeFunctions[index].Func)
		funcName = fn.NativeFunctions[index].Name
	} else {
		funcType = fn.ScrigoFunctions[index].Type
		funcName = fn.ScrigoFunctions[index].Name
	}
	print := func(t reflect.Type) string {
		str := ""
		switch kindToType(t.Kind()) {
		case vm.TypeInt:
			stackShift[0]++
			str += fmt.Sprintf("i%d %v", stackShift[0], t)
		case vm.TypeFloat:
			stackShift[1]++
			str += fmt.Sprintf("f%d %v", stackShift[1], t)
		case vm.TypeString:
			stackShift[2]++
			str += fmt.Sprintf("s%d %v", stackShift[2], t)
		case vm.TypeGeneral:
			stackShift[3]++
			str += fmt.Sprintf("g%d %v", stackShift[3], t)
		}
		return str
	}
	out := ""
	for i := 0; i < funcType.NumOut(); i++ {
		out += print(funcType.Out(i))
		if i < funcType.NumOut()-1 {
			out += ", "
		}
	}
	in := ""
	for i := 0; i < funcType.NumIn()-1; i++ {
		in += print(funcType.In(i)) + ", "
	}
	if funcType.NumIn()-1 >= 0 {
		if variadic == vm.NoVariadic || variadic == 0 {
			in += print(funcType.In(funcType.NumIn() - 1))
		} else {
			varType := funcType.In(funcType.NumIn() - 1).Elem()
			for i := int8(0); i < variadic; i++ {
				in += print(varType)
				if i < variadic-1 {
					in += ", "
				}
			}
		}
	}
	return fmt.Sprintf("%s(%s) (%s)", funcName, in, out)
}

func disassembleVarRef(fn *vm.ScrigoFunction, ref int16) string {
	depth := 0
	for ref >= 0 && fn.Parent != nil {
		ref = fn.VarRefs[ref]
		depth++
		fn = fn.Parent
	}
	if depth == 0 {
		v := fn.Globals[ref]
		return packageName(v.Pkg) + "." + v.Name
	}
	s := disassembleOperand(fn, -int8(ref), vm.Interface, false)
	if depth > 0 {
		s += "@" + strconv.Itoa(depth)
	}
	return s
}

func reflectToRegisterKind(kind reflect.Kind) vm.Kind {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return vm.Int
	case reflect.Bool:
		return vm.Bool
	case reflect.Float32, reflect.Float64:
		return vm.Float64
	case reflect.String:
		return vm.String
	default:
		return vm.Interface
	}
}

func registerKindToLabel(kind vm.Kind) string {
	switch kind {
	case vm.Bool, vm.Int, vm.Int8, vm.Int16, vm.Int32, vm.Int64,
		vm.Uint, vm.Uint8, vm.Uint16, vm.Uint32, vm.Uint64:
		return "i"
	case vm.Float32, vm.Float64:
		return "f"
	case vm.String:
		return "s"
	default:
		return "g"
	}
}

func disassembleOperand(fn *vm.ScrigoFunction, op int8, kind vm.Kind, constant bool) string {
	if constant {
		switch {
		case vm.Int <= kind && kind <= vm.Uint64:
			return strconv.Itoa(int(op))
		case kind == vm.Float64:
			return strconv.FormatFloat(float64(op), 'f', -1, 64)
		case kind == vm.Float32:
			return strconv.FormatFloat(float64(op), 'f', -1, 32)
		case kind == vm.Bool:
			if op == 0 {
				return "false"
			}
			return "true"
		case kind == vm.String:
			return strconv.Quote(fn.Constants.String[uint8(op)])
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

	vm.OpNone: "Nop", // TODO(Gianluca): review.

	vm.OpAddInt:     "Add",
	vm.OpAddInt8:    "Add8",
	vm.OpAddInt16:   "Add16",
	vm.OpAddInt32:   "Add32",
	vm.OpAddFloat32: "Add32",
	vm.OpAddFloat64: "Add",

	vm.OpAnd: "And",

	vm.OpAndNot: "AndNot",

	vm.OpAppend: "Append",

	vm.OpAppendSlice: "AppendSlice",

	vm.OpAssert: "Assert",

	vm.OpBind: "Bind",

	vm.OpBreak: "Break",

	vm.OpCall: "Call",

	vm.OpCallIndirect: "CallIndirect",

	vm.OpCallNative: "CallNative",

	vm.OpCap: "Cap",

	vm.OpCase: "Case",

	vm.OpClose: "Close",

	vm.OpContinue: "Continue",

	vm.OpConvert:       "Convert",
	vm.OpConvertInt:    "Convert",
	vm.OpConvertUint:   "ConvertU",
	vm.OpConvertFloat:  "Convert",
	vm.OpConvertString: "Convert",

	vm.OpCopy: "Copy",

	vm.OpConcat: "concat",

	vm.OpDefer: "Defer",

	vm.OpDelete: "delete",

	vm.OpDivInt:     "Div",
	vm.OpDivInt8:    "Div8",
	vm.OpDivInt16:   "Div16",
	vm.OpDivInt32:   "Div32",
	vm.OpDivUint8:   "DivU8",
	vm.OpDivUint16:  "DivU16",
	vm.OpDivUint32:  "DivU32",
	vm.OpDivUint64:  "DivU64",
	vm.OpDivFloat32: "Div32",
	vm.OpDivFloat64: "Div",

	vm.OpFunc: "Func",

	vm.OpGetFunc: "GetFunc",

	vm.OpGetVar: "GetVar",

	vm.OpGo: "Go",

	vm.OpGoto: "Goto",

	vm.OpIf:       "If",
	vm.OpIfInt:    "If",
	vm.OpIfUint:   "IfU",
	vm.OpIfFloat:  "If",
	vm.OpIfString: "If",

	vm.OpIndex: "Index",

	vm.OpLeftShift:   "LeftShift",
	vm.OpLeftShift8:  "LeftShift8",
	vm.OpLeftShift16: "LeftShift16",
	vm.OpLeftShift32: "LeftShift32",

	vm.OpLen: "Len",

	vm.OpLoadNumber: "LoadNumber",

	vm.OpMakeChan: "MakeChan",

	vm.OpMapIndex: "MapIndex",

	vm.OpMakeMap: "MakeMap",

	vm.OpMakeSlice: "MakeSlice",

	vm.OpMove: "Move",

	vm.OpMulInt:     "Mul",
	vm.OpMulInt8:    "Mul8",
	vm.OpMulInt16:   "Mul16",
	vm.OpMulInt32:   "Mul32",
	vm.OpMulFloat32: "Mul32",
	vm.OpMulFloat64: "Mul",

	vm.OpNew: "New",

	vm.OpOr: "Or",

	vm.OpPanic: "Panic",

	vm.OpPrint: "Print",

	vm.OpRange: "Range",

	vm.OpRangeString: "Range",

	vm.OpReceive: "Receive",

	vm.OpRecover: "Recover",

	vm.OpRemInt:    "Rem",
	vm.OpRemInt8:   "Rem8",
	vm.OpRemInt16:  "Rem16",
	vm.OpRemInt32:  "Rem32",
	vm.OpRemUint8:  "RemU8",
	vm.OpRemUint16: "RemU16",
	vm.OpRemUint32: "RemU32",
	vm.OpRemUint64: "RemU64",

	vm.OpReturn: "Return",

	vm.OpRightShift:  "RightShift",
	vm.OpRightShiftU: "RightShiftU",

	vm.OpSelect: "Select",

	vm.OpSelector: "Selector",

	vm.OpSend: "Send",

	vm.OpSetMap: "SetMap",

	vm.OpSetSlice: "SetSlice",

	vm.OpSetVar: "SetVar",

	vm.OpSliceIndex: "SliceIndex",

	vm.OpStringIndex: "StringIndex",

	vm.OpSubInt:     "Sub",
	vm.OpSubInt8:    "Sub8",
	vm.OpSubInt16:   "Sub16",
	vm.OpSubInt32:   "Sub32",
	vm.OpSubFloat32: "Sub32",
	vm.OpSubFloat64: "Sub",

	vm.OpSubInvInt:     "SubInv",
	vm.OpSubInvInt8:    "SubInv8",
	vm.OpSubInvInt16:   "SubInv16",
	vm.OpSubInvInt32:   "SubInv32",
	vm.OpSubInvFloat32: "SubInv32",
	vm.OpSubInvFloat64: "SubInv",

	vm.OpTailCall: "TailCall",

	vm.OpXor: "Xor",
}

var conditionName = [...]string{
	vm.ConditionEqual:             "Equal",
	vm.ConditionNotEqual:          "NotEqual",
	vm.ConditionLess:              "Less",
	vm.ConditionLessOrEqual:       "LessOrEqual",
	vm.ConditionGreater:           "Greater",
	vm.ConditionGreaterOrEqual:    "GreaterOrEqual",
	vm.ConditionEqualLen:          "EqualLen",
	vm.ConditionNotEqualLen:       "NotEqualLen",
	vm.ConditionLessLen:           "LessLen",
	vm.ConditionLessOrEqualLen:    "LessOrEqualLen",
	vm.ConditionGreaterLen:        "GreaterOrEqualLen",
	vm.ConditionGreaterOrEqualLen: "GreaterOrEqualLen",
	vm.ConditionNil:               "Nil",
	vm.ConditionNotNil:            "NotNil",
	vm.ConditionOK:                "OK",
	vm.ConditionNotOK:             "NotOK",
}
