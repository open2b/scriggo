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

func Disassemble(main *ScrigoFunction) (assembler map[string]string, err error) {

	functionsByPkg := map[string]map[*ScrigoFunction]int{}
	importsByPkg := map[string]map[string]struct{}{}

	c := len(main.ScrigoFunctions)
	if c == 0 {
		c = 1
	}
	allFunctions := make([]*ScrigoFunction, 1, c)
	allFunctions[0] = main

	for i := 0; i < len(allFunctions); i++ {
		fn := allFunctions[i]
		if p, ok := functionsByPkg[fn.Pkg]; ok {
			p[fn] = fn.Line
		} else {
			functionsByPkg[fn.Pkg] = map[*ScrigoFunction]int{fn: fn.Line}
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

		functions := make([]*ScrigoFunction, 0, len(funcs))
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

func DisassembleFunction(w io.Writer, fn *ScrigoFunction) (int64, error) {
	var b bytes.Buffer
	_, _ = fmt.Fprintf(&b, "Func %s", fn.Name)
	disassembleFunction(&b, fn, 0)
	return b.WriteTo(w)
}

func disassembleFunction(w *bytes.Buffer, fn *ScrigoFunction, depth int) {
	indent := ""
	if depth > 0 {
		indent = strings.Repeat("\t", depth)
	}
	labelOf := map[uint32]uint32{}
	for _, in := range fn.Body {
		if in.Op == opGoto {
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
		case opGoto:
			label := labelOf[decodeAddr(in.A, in.B, in.C)]
			_, _ = fmt.Fprintf(w, "%s\tGoto %d\n", indent, label)
		case opFunc:
			_, _ = fmt.Fprintf(w, "%s\tFunc %s ", indent, disassembleOperand(fn, in.C, Interface, false))
			disassembleFunction(w, fn.Literals[uint8(in.B)], depth+1)
		default:
			_, _ = fmt.Fprintf(w, "%s\t%s\n", indent, disassembleInstruction(fn, addr))
		}
		switch in.Op {
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
	in := fn.Body[addr]
	op, a, b, c := in.Op, in.A, in.B, in.C
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
		opSubInvInt, opSubInvInt8, opSubInvInt16, opSubInvInt32,
		opLeftShift, opLeftShift8, opLeftShift16, opLeftShift32,
		opRightShift, opRightShiftU:
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
		s += " type(" + fn.Types[b].String() + ")"
		t := fn.Types[int(uint(b))]
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
		cv := fn.CRefs[uint8(b)]
		var depth = 1
		for p := fn.Parent; cv >= 0; p = p.Parent {
			cv = p.CRefs[cv]
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
				sf := fn.ScrigoFunctions[uint8(a)]
				s += " " + packageName(sf.Pkg) + "." + sf.Name
			case opCallNative:
				nf := fn.NativeFunctions[uint8(a)]
				s += " " + packageName(nf.Pkg) + "." + nf.Name
			case opCallIndirect, opDefer:
				s += " " + disassembleOperand(fn, a, Interface, false)
			}
		}
		if c != NoVariadic && (op == opCallIndirect || op == opCallNative || op == opDefer) {
			s += " ..." + strconv.Itoa(int(c))
		}
		switch op {
		case opCall, opCallIndirect, opCallNative, opDefer:
			grow := fn.Body[addr+1]
			s += "\t// Stack shift: " + strconv.Itoa(int(grow.Op)) + ", " + strconv.Itoa(int(grow.A)) + ", " +
				strconv.Itoa(int(grow.B)) + ", " + strconv.Itoa(int(grow.C))
		}
		if op == opDefer {
			args := fn.Body[addr+2]
			s += "; args: " + strconv.Itoa(int(args.Op)) + ", " + strconv.Itoa(int(args.A)) + ", " +
				strconv.Itoa(int(args.B)) + ", " + strconv.Itoa(int(args.C))

		}
	case opCap:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " " + disassembleOperand(fn, c, Int, false)
	case opCase:
		switch reflect.SelectDir(a) {
		case reflect.SelectSend:
			s += " send " + disassembleOperand(fn, b, Int, k) + " " + disassembleOperand(fn, c, Interface, false)
		case reflect.SelectRecv:
			s += " recv " + disassembleOperand(fn, b, Int, false) + " " + disassembleOperand(fn, c, Interface, false)
		default:
			s += " default"
		}
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
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " " + fn.Types[int(uint(b))].String()
		s += " " + disassembleOperand(fn, c, Interface, false)
	case opConvertInt, opConvertUint:
		s += " " + disassembleOperand(fn, a, Int, false)
		s += " " + fn.Types[int(uint(b))].String()
		s += " " + disassembleOperand(fn, c, Interface, false)
	case opConvertFloat:
		s += " " + disassembleOperand(fn, a, Float64, false)
		s += " " + fn.Types[int(uint(b))].String()
		s += " " + disassembleOperand(fn, c, Interface, false)
	case opConvertString:
		s += " " + disassembleOperand(fn, a, String, false)
		s += " " + fn.Types[int(uint(b))].String()
		s += " " + disassembleOperand(fn, c, Interface, false)
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
		if a == 0 {
			f := fn.ScrigoFunctions[uint8(b)]
			s += " " + packageName(f.Pkg) + "." + f.Name
		} else {
			f := fn.NativeFunctions[uint8(b)]
			s += " " + packageName(f.Pkg) + "." + f.Name
		}
		s += " " + disassembleOperand(fn, c, Interface, false)
	case opGetVar:
		s += " " + packageName(fn.Pkg) + "."
		name := fn.Variables[uint8(b)].Name
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
	case opLoadNumber:
		if a == 0 {
			s += " int"
			s += " " + fmt.Sprintf("%d", fn.Constants.Int[uint8(b)])
			s += " " + disassembleOperand(fn, c, Int, false)
		} else {
			s += " float"
			s += " " + fmt.Sprintf("%f", fn.Constants.Float[uint8(b)])
			s += " " + disassembleOperand(fn, c, Float64, false)
		}

	case opMakeChan:
		s += " " + fn.Types[int(uint(a))].String()
		s += " " + disassembleOperand(fn, b, Int, k)
		s += " " + disassembleOperand(fn, c, Interface, false)
	case opMakeMap:
		s += " " + fn.Types[int(uint(a))].String()
		s += " " + disassembleOperand(fn, b, Int, k)
		s += " " + disassembleOperand(fn, c, Interface, false)
	case opMapIndex:
		//s += " " + disassembleOperand(scrigo, a, Interface, false)
		//key := reflectToRegisterKind()
		//s += " " + disassembleOperand(scrigo, b, Interface, false)
		//s += " " + disassembleOperand(scrigo, c, Interface, false)
	case opMove:
		switch MoveType(a) {
		case IntInt:
			s += " " + disassembleOperand(fn, b, Int, k)
			s += " " + disassembleOperand(fn, c, Int, false)
		case FloatFloat:
			s += " " + disassembleOperand(fn, b, Float64, k)
			s += " " + disassembleOperand(fn, c, Float64, false)
		case StringString:
			s += " " + disassembleOperand(fn, b, String, k)
			s += " " + disassembleOperand(fn, c, String, false)
		case GeneralGeneral:
			s += " " + disassembleOperand(fn, b, Interface, k)
			s += " " + disassembleOperand(fn, c, Interface, false)
		case IntGeneral:
			s += " " + disassembleOperand(fn, b, Int, k)
			s += " " + disassembleOperand(fn, c, Interface, false)
		case FloatGeneral:
			s += " " + disassembleOperand(fn, b, Float64, k)
			s += " " + disassembleOperand(fn, c, Interface, false)
		case StringGeneral:
			s += " " + disassembleOperand(fn, b, String, k)
			s += " " + disassembleOperand(fn, c, Interface, false)
		}
	case opNew:
		s += " " + fn.Types[int(uint(b))].String()
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
		s += " " + fn.Types[int(uint(a))].String()
		s += " " + disassembleOperand(fn, c, Int, false)
		s += " // len: "
		s += fmt.Sprintf("%d", fn.Body[addr+1].A)
		s += ", cap: "
		s += fmt.Sprintf("%d", fn.Body[addr+1].B)
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
		s += " " + disassembleOperand(fn, c, Int, false)
	case opSetVar:
		s += " " + disassembleOperand(fn, b, Interface, false)
		v := fn.Variables[uint8(c)]
		s += " " + packageName(v.Name) + "."
		if v.Name == "" {
			s += strconv.Itoa(int(uint8(c)))
		} else {
			s += v.Name
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

func registerKindToLabel(kind Kind) string {
	switch kind {
	case Bool, Int, Int8, Int16, Int32, Int64,
		Uint, Uint8, Uint16, Uint32, Uint64:
		return "i"
	case Float32, Float64:
		return "f"
	case String:
		return "s"
	default:
		return "g"
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

func (op operation) String() string {
	return operationName[op]
}

var operationName = [...]string{

	opNone: "Nop", // TODO(Gianluca): review.

	opAddInt:     "Add",
	opAddInt8:    "Add8",
	opAddInt16:   "Add16",
	opAddInt32:   "Add32",
	opAddFloat32: "Add32",
	opAddFloat64: "Add",

	opAnd: "And",

	opAndNot: "AndNot",

	opAppend: "Append",

	opAppendSlice: "AppendSlice",

	opAssert:        "Assert",
	opAssertInt:     "Assert",
	opAssertFloat64: "Assert",
	opAssertString:  "Assert",

	opBind: "Bind",

	opCall: "Call",

	opCallIndirect: "CallIndirect",

	opCallNative: "CallNative",

	opCap: "Cap",

	opCase: "Case",

	opContinue: "Continue",

	opConvert:       "Convert",
	opConvertInt:    "Convert",
	opConvertUint:   "ConvertU",
	opConvertFloat:  "Convert",
	opConvertString: "Convert",

	opCopy: "Copy",

	opConcat: "concat",

	opDefer: "Defer",

	opDelete: "delete",

	opDivInt:     "Div",
	opDivInt8:    "Div8",
	opDivInt16:   "Div16",
	opDivInt32:   "Div32",
	opDivUint8:   "DivU8",
	opDivUint16:  "DivU16",
	opDivUint32:  "DivU32",
	opDivUint64:  "DivU64",
	opDivFloat32: "Div32",
	opDivFloat64: "Div",

	opFunc: "Func",

	opGetFunc: "GetFunc",

	opGetVar: "GetVar",

	opGo: "Go",

	opGoto: "Goto",

	opIf:       "If",
	opIfInt:    "If",
	opIfUint:   "IfU",
	opIfFloat:  "If",
	opIfString: "If",

	opIndex: "Index",

	opLeftShift:   "LeftShift",
	opLeftShift8:  "LeftShift8",
	opLeftShift16: "LeftShift16",
	opLeftShift32: "LeftShift32",

	opLen: "Len",

	opLoadNumber: "LoadNumber",

	opMakeChan: "MakeChan",

	opMapIndex: "MapIndex",

	opMakeMap: "MakeMap",

	opMakeSlice: "MakeSlice",

	opMove: "Move",

	opMulInt:     "Mul",
	opMulInt8:    "Mul8",
	opMulInt16:   "Mul16",
	opMulInt32:   "Mul32",
	opMulFloat32: "Mul32",
	opMulFloat64: "Mul",

	opNew: "New",

	opOr: "Or",

	opPanic: "Panic",

	opPrint: "Print",

	opRange: "Range",

	opRangeString: "Range",

	opReceive: "Receive",

	opRecover: "Recover",

	opRemInt:    "Rem",
	opRemInt8:   "Rem8",
	opRemInt16:  "Rem16",
	opRemInt32:  "Rem32",
	opRemUint8:  "RemU8",
	opRemUint16: "RemU16",
	opRemUint32: "RemU32",
	opRemUint64: "RemU64",

	opReturn: "Return",

	opRightShift:  "RightShift",
	opRightShiftU: "RightShiftU",

	opSelect: "Select",

	opSelector: "Selector",

	opSend: "Send",

	opSetMap: "SetMap",

	opSetSlice: "SetSlice",

	opSetVar: "SetPackageVar",

	opSliceIndex: "SliceIndex",

	opStringIndex: "StringIndex",

	opSubInt:     "Sub",
	opSubInt8:    "Sub8",
	opSubInt16:   "Sub16",
	opSubInt32:   "Sub32",
	opSubFloat32: "Sub32",
	opSubFloat64: "Sub",

	opSubInvInt:     "SubInv",
	opSubInvInt8:    "SubInv8",
	opSubInvInt16:   "SubInv16",
	opSubInvInt32:   "SubInv32",
	opSubInvFloat32: "SubInv32",
	opSubInvFloat64: "SubInv",

	opTailCall: "TailCall",

	opXor: "Xor",
}
