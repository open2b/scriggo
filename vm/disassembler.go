// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

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
)

func packageName(pkg string) string {
	i := strings.LastIndex(pkg, "/")
	return pkg[i+1:]
}

func DisassembleDir(dir string, main *ScrigoFunction) (err error) {

	packages, err := Disassemble(main)
	if err != nil {
		return err
	}

	for path, source := range packages {
		pkgDir, file := filepath.Split(path)
		fullDir := filepath.Join(dir, pkgDir)
		err = os.MkdirAll(fullDir, 0775)
		if err != nil {
			return err
		}
		fi, err := os.OpenFile(filepath.Join(fullDir, file+".s"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
		if err != nil {
			return err
		}
		defer func() {
			err = fi.Close()
		}()
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

func Disassemble(main *ScrigoFunction) (assembler map[string]string, err error) {

	functionsByPkg := map[string]map[*ScrigoFunction]int{}
	importsByPkg := map[string]map[string]struct{}{}

	c := len(main.scrigoFunctions)
	if c == 0 {
		c = 1
	}
	allFunctions := make([]*ScrigoFunction, 1, c)
	allFunctions[0] = main

	for i := 0; i < len(allFunctions); i++ {
		fn := allFunctions[i]
		if p, ok := functionsByPkg[fn.pkg]; ok {
			p[fn] = fn.line
		} else {
			functionsByPkg[fn.pkg] = map[*ScrigoFunction]int{fn: fn.line}
		}
		for _, sf := range fn.scrigoFunctions {
			if sf.pkg != fn.pkg {
				if packages, ok := importsByPkg[fn.pkg]; ok {
					packages[sf.pkg] = struct{}{}
				} else {
					importsByPkg[fn.pkg] = map[string]struct{}{sf.pkg: {}}
				}
			}
		}
		for _, nf := range fn.nativeFunctions {
			if packages, ok := importsByPkg[fn.pkg]; ok {
				packages[nf.pkg] = struct{}{}
			} else {
				importsByPkg[fn.pkg] = map[string]struct{}{nf.pkg: {}}
			}
		}
		allFunctions = append(allFunctions, fn.scrigoFunctions...)
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

		functions := make([]*ScrigoFunction, 0, len(funcs))
		for fn := range funcs {
			functions = append(functions, fn)
		}
		sort.Slice(functions, func(i, j int) bool { return functions[i].line < functions[i].line })

		for _, fn := range functions {
			_, _ = b.WriteString("\nFunc ")
			_, _ = b.WriteString(fn.name)
			disassembleFunction(&b, fn, 0)
		}
		_, _ = b.WriteRune('\n')

		assembler[path] = b.String()

		b.Reset()

	}

	return assembler, nil
}

func DisassembleFunction(w io.Writer, fn *ScrigoFunction) (int64, error) {
	var b bytes.Buffer
	_, _ = fmt.Fprintf(&b, "Func %s", fn.name)
	disassembleFunction(&b, fn, 0)
	return b.WriteTo(w)
}

func disassembleFunction(w *bytes.Buffer, fn *ScrigoFunction, depth int) {
	indent := ""
	if depth > 0 {
		indent = strings.Repeat("\t", depth)
	}
	labelOf := map[uint32]uint32{}
	for _, in := range fn.body {
		if in.op == opGoto {
			labelOf[decodeAddr(in.a, in.b, in.c)] = 0
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
	nIn := fn.typ.NumIn()
	nOut := fn.typ.NumOut()
	for i := 0; i < nIn; i++ {
		if i > 0 {
			_, _ = fmt.Fprint(w, ", ")
		}
		_, _ = fmt.Fprintf(w, "%d %s", nOut+i+1, fn.typ.In(i))
	}
	_, _ = fmt.Fprint(w, ")")
	if nOut > 0 {
		_, _ = fmt.Fprint(w, " (")
		for i := 0; i < nOut; i++ {
			if i > 0 {
				_, _ = fmt.Fprint(w, ", ")
			}
			_, _ = fmt.Fprintf(w, "%d %s", i+1, fn.typ.Out(i))
		}
		_, _ = fmt.Fprint(w, ")")
	}
	_, _ = fmt.Fprint(w, "\n")
	_, _ = fmt.Fprintf(w, "%s\t// regs(%d,%d,%d,%d)\n", indent,
		fn.regnum[0], fn.regnum[1], fn.regnum[2], fn.regnum[3])
	instrNum := uint32(len(fn.body))
	for addr := uint32(0); addr < instrNum; addr++ {
		if label, ok := labelOf[uint32(addr)]; ok {
			_, _ = fmt.Fprintf(w, "%s%d:", indent, label)
		}
		in := fn.body[addr]
		switch in.op {
		case opGoto:
			label := labelOf[decodeAddr(in.a, in.b, in.c)]
			_, _ = fmt.Fprintf(w, "%s\tGoto %d\n", indent, label)
		case opFunc:
			_, _ = fmt.Fprintf(w, "%s\tFunc %s ", indent, disassembleOperand(fn, in.c, Interface, false))
			disassembleFunction(w, fn.literals[uint8(in.b)], depth+1)
		default:
			_, _ = fmt.Fprintf(w, "%s\t%s\n", indent, disassembleInstruction(fn, addr))
		}
		switch in.op {
		case opCall, opCallIndirect, opCallNative, opTailCall, opMakeSlice:
			addr += 1
		case opDefer:
			addr += 2
		}
	}
}

func DisassembleInstruction(w io.Writer, fn *ScrigoFunction, addr uint32) (int64, error) {
	n, err := io.WriteString(w, disassembleInstruction(fn, addr))
	return int64(n), err
}

func disassembleInstruction(fn *ScrigoFunction, addr uint32) string {
	in := fn.body[addr]
	op, a, b, c := in.op, in.a, in.b, in.c
	k := false
	if op < 0 {
		op = -op
		k = true
	}
	s := op.String()
	switch op {
	case opAddInt, opAddInt8, opAddInt16, opAddInt32,
		opAnd, opAndNot, opOr, opXor,
		opDivInt, opDivInt8, opDivInt16, opDivInt32, opDivUint8, opDivUint16, opDivUint32, opDivUint64,
		opMulInt, opMulInt8, opMulInt16, opMulInt32,
		opRemInt, opRemInt8, opRemInt16, opRemInt32, opRemUint8, opRemUint16, opRemUint32, opRemUint64,
		opSubInt, opSubInt8, opSubInt16, opSubInt32,
		opSubInvInt, opSubInvInt8, opSubInvInt16, opSubInvInt32:
		s += " " + disassembleOperand(fn, a, Int, false)
		s += " " + disassembleOperand(fn, b, Int, k)
		s += " " + disassembleOperand(fn, c, Int, false)
	case opAddFloat32, opAddFloat64, opDivFloat32, opDivFloat64,
		opMulFloat32, opMulFloat64,
		opSubFloat32, opSubFloat64, opSubInvFloat32, opSubInvFloat64:
		s += " " + disassembleOperand(fn, a, Float64, false)
		s += " " + disassembleOperand(fn, b, Float64, k)
		s += " " + disassembleOperand(fn, c, Float64, false)
	case opAssert:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " type(" + strconv.Itoa(int(uint(b))) + ")"
		t := fn.types[int(uint(b))]
		var kind = reflectToRegisterKind(t.Kind())
		s += " " + disassembleOperand(fn, c, kind, false)
	case opAssertInt:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " type(int)"
		s += " " + disassembleOperand(fn, c, Int, false)
	case opAssertFloat64:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " type(float64)"
		s += " " + disassembleOperand(fn, c, Float64, false)
	case opAssertString:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " type(string)"
		s += " " + disassembleOperand(fn, c, String, false)
	case opBind:
		cv := fn.crefs[uint8(b)]
		var depth = 1
		for p := fn.parent; cv >= 0; p = p.parent {
			cv = p.crefs[cv]
			depth++
		}
		s += " " + disassembleOperand(fn, -int8(cv), Interface, false)
		if depth > 0 {
			s += "@" + strconv.Itoa(depth)
		}
		s += " " + disassembleOperand(fn, c, Int, false)
	case opCall, opCallIndirect, opCallNative, opTailCall, opDefer:
		if a != CurrentFunction {
			switch op {
			case opCall, opTailCall:
				sf := fn.scrigoFunctions[uint8(b)]
				s += " " + packageName(sf.pkg) + "." + sf.name
			case opCallNative:
				nf := fn.nativeFunctions[uint8(b)]
				s += " " + packageName(nf.pkg) + "." + nf.name
			case opCallIndirect, opDefer:
				s += " " + disassembleOperand(fn, a, Interface, false)
			}
		}
		if c != NoVariadic && (op == opCallIndirect || op == opCallNative || op == opDefer) {
			s += " ..." + strconv.Itoa(int(c))
		}
		switch op {
		case opCall, opCallIndirect, opCallNative, opDefer:
			grow := fn.body[addr+1]
			s += "\t// Stack shift: " + strconv.Itoa(int(grow.op)) + ", " + strconv.Itoa(int(grow.a)) + ", " +
				strconv.Itoa(int(grow.b)) + ", " + strconv.Itoa(int(grow.c))
		}
		if op == opDefer {
			args := fn.body[addr+2]
			s += "; args: " + strconv.Itoa(int(args.op)) + ", " + strconv.Itoa(int(args.a)) + ", " +
				strconv.Itoa(int(args.b)) + ", " + strconv.Itoa(int(args.c))

		}
	case opCap:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " " + disassembleOperand(fn, c, Int, false)
	case opContinue:
		s += " " + disassembleOperand(fn, b, Int, false)
	case opCopy:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " " + disassembleOperand(fn, b, Interface, false)
		s += " " + disassembleOperand(fn, c, Int, false)
	case opConcat:
		s += " " + disassembleOperand(fn, a, String, false)
		s += " " + disassembleOperand(fn, b, String, k)
		s += " " + disassembleOperand(fn, c, String, false)
	case opConvert:
		s += " " + disassembleOperand(fn, a, String, false)
		s += " " + fn.types[int(uint(b))].String()
		s += " " + disassembleOperand(fn, c, String, false)
	case opDelete:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " " + disassembleOperand(fn, b, Interface, false)
	case opIf:
		if Condition(b) == ConditionOK {
			s += " OK"
		} else {
			s += " " + disassembleOperand(fn, a, Interface, false)
			s += " " + Condition(b).String()
		}
	case opIfInt, opIfUint:
		s += " " + disassembleOperand(fn, a, Int, false)
		s += " " + Condition(b).String()
		if Condition(b) >= ConditionEqual {
			s += " " + disassembleOperand(fn, c, Int, k)
		}
	case opIfFloat:
		s += " " + disassembleOperand(fn, a, Float64, false)
		s += " " + Condition(b).String()
		s += " " + disassembleOperand(fn, c, Float64, k)
	case opIfString:
		s += " " + disassembleOperand(fn, a, String, false)
		s += " " + Condition(b).String()
		if Condition(b) < ConditionEqualLen {
			if k && c >= 0 {
				s += " " + strconv.Quote(string(c))
			} else {
				s += " " + disassembleOperand(fn, c, String, k)
			}
		} else {
			s += " " + disassembleOperand(fn, c, Int, k)
		}
	case opFunc:
		s += " func(" + strconv.Itoa(int(uint8(b))) + ")"
		s += " " + disassembleOperand(fn, c, Int, false)
	case opGetFunc:
		s += " " + packageName(fn.pkg) + "." + fn.scrigoFunctions[uint8(b)].name
		s += " " + disassembleOperand(fn, c, Interface, false)
	case opGetVar:
		s += " " + packageName(fn.pkg) + "."
		name := fn.variables[uint8(b)].name
		if name == "" {
			s += strconv.Itoa(int(uint8(b)))
		} else {
			s += name
		}
		s += " " + disassembleOperand(fn, c, Interface, false)
	case opGo, opReturn:
	case opGoto:
		s += " " + strconv.Itoa(int(decodeAddr(a, b, c)))
	case opIndex:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " " + disassembleOperand(fn, b, Int, k)
		s += " " + disassembleOperand(fn, c, Interface, false)
	case opLen:
		s += " " + strconv.Itoa(int(a))
		if a == 0 {
			s += " " + disassembleOperand(fn, b, String, false)
		} else {
			s += " " + disassembleOperand(fn, b, Interface, false)
		}
		s += " " + disassembleOperand(fn, c, Int, false)
	case opMakeChan:
		s += " type(" + strconv.Itoa(int(uint(a))) + ")"
		s += " " + disassembleOperand(fn, b, Int, false)
		s += " " + disassembleOperand(fn, c, Interface, false)
	case opMakeMap:
		s += " " + fn.types[int(uint(a))].String()
		s += " " + disassembleOperand(fn, b, Int, k)
		s += " " + disassembleOperand(fn, c, Interface, false)
	case opMapIndex:
		//s += " " + disassembleOperand(scrigo, a, Interface, false)
		//key := reflectToRegisterKind()
		//s += " " + disassembleOperand(scrigo, b, Interface, false)
		//s += " " + disassembleOperand(scrigo, c, Interface, false)
	case opMapIndexStringString:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " " + disassembleOperand(fn, b, String, k)
		s += " " + disassembleOperand(fn, c, String, false)
	case opMapIndexStringInt:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " " + disassembleOperand(fn, b, String, k)
		s += " " + disassembleOperand(fn, c, String, false)
	case opMapIndexStringInterface:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " " + disassembleOperand(fn, b, String, k)
		s += " " + disassembleOperand(fn, c, Interface, false)
	case opMove:
		switch MoveType(a) {
		case IntInt:
			s += " int"
			s += " " + disassembleOperand(fn, b, Int, k)
			s += " " + disassembleOperand(fn, c, Int, false)
		case FloatFloat:
			s += " float64"
			s += " " + disassembleOperand(fn, b, Float64, k)
			s += " " + disassembleOperand(fn, c, Float64, false)
		case StringString:
			s += " string"
			s += " " + disassembleOperand(fn, b, String, k)
			s += " " + disassembleOperand(fn, c, String, false)
		case GeneralGeneral:
			s += " " + disassembleOperand(fn, b, Interface, k)
			s += " " + disassembleOperand(fn, c, Interface, false)
		case IntGeneral:
			s += " intToGeneral"
			s += " " + disassembleOperand(fn, b, Int, k)
			s += " " + disassembleOperand(fn, c, Interface, false)
		case FloatGeneral:
			s += " floatToGeneral"
			s += " " + disassembleOperand(fn, b, Float64, k)
			s += " " + disassembleOperand(fn, c, Interface, false)
		case StringGeneral:
			s += " stringToGeneral"
			s += " " + disassembleOperand(fn, b, String, k)
			s += " " + disassembleOperand(fn, c, Interface, false)
		}
	case opNew:
		s += " " + fn.types[int(uint(b))].String()
		s += " " + disassembleOperand(fn, c, Interface, false)
	case opPanic, opPrint:
		s += " " + disassembleOperand(fn, a, Interface, false)
	case opRange:
		//s += " " + disassembleOperand(scrigo, c, Interface, false)
	case opRangeString:
		s += " " + disassembleOperand(fn, a, Int, false)
		s += " " + disassembleOperand(fn, b, Int, false)
		s += " " + disassembleOperand(fn, c, Interface, k)
	case opReceive:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " " + disassembleOperand(fn, b, Bool, false)
		s += " " + disassembleOperand(fn, c, Interface, false)
	case opRecover:
		if c != 0 {
			s += " " + disassembleOperand(fn, c, Interface, false)
		}
	case opSelector:
		//s += " " + disassembleOperand(scrigo, c, Interface, false)
	case opSend:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " " + disassembleOperand(fn, c, Interface, false)
	case opMakeSlice:
		s += " " + fn.types[int(uint(a))].String()
		s += " " + fmt.Sprintf("0b%b", b)
		s += " " + disassembleOperand(fn, c, Int, false)
		s += " // len: "
		s += fmt.Sprintf("%d", fn.body[addr+1].a)
		s += ", cap: "
		s += fmt.Sprintf("%d", fn.body[addr+1].b)
	case opSetMap:
		s += " " + disassembleOperand(fn, a, Interface, false)
		if k {
			s += fmt.Sprintf(" K(%v)", b)
		} else {
			s += " " + disassembleOperand(fn, b, Interface, false)
		}
		s += " " + disassembleOperand(fn, c, Interface, false)
	case opSetSlice:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " " + disassembleOperand(fn, b, Int, k)
		s += " " + disassembleOperand(fn, c, Interface, false)
	case opSetVar:
		s += " " + disassembleOperand(fn, b, Interface, false)
		v := fn.variables[uint8(c)]
		s += " " + packageName(v.name) + "."
		if v.name == "" {
			s += strconv.Itoa(int(uint8(c)))
		} else {
			s += v.name
		}
	case opSliceIndex:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " " + disassembleOperand(fn, b, Int, k)
		//s += " " + disassembleOperand(scrigo, c, Interface, false)
	}
	return s
}

func reflectToRegisterKind(kind reflect.Kind) Kind {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return Int
	case reflect.Bool:
		return Bool
	case reflect.Float32, reflect.Float64:
		return Float64
	case reflect.String:
		return String
	default:
		return Interface
	}
}

func disassembleOperand(fn *ScrigoFunction, op int8, kind Kind, constant bool) string {
	if constant {
		switch {
		case Int <= kind && kind <= Uint64:
			return strconv.Itoa(int(op))
		case kind == Float64:
			return strconv.FormatFloat(float64(op), 'f', -1, 64)
		case kind == Float32:
			return strconv.FormatFloat(float64(op), 'f', -1, 32)
		case kind == Bool:
			if op == 0 {
				return "false"
			}
			return "true"
		case kind == String:
			return strconv.Quote(fn.constants.String[uint8(op)])
		default:
			v := fn.constants.General[uint8(op)]
			if v == nil {
				return "nil"
			}
			return fmt.Sprintf("%#v", v)
		}
	}
	if op > 0 {
		return "R" + strconv.Itoa(int(op))
	}
	if op == 0 {
		return "_"
	}
	return "(R" + strconv.Itoa(-int(op)) + ")"
}
