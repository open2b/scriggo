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
		if in.Op == OpGoto {
			labelOf[DecodeAddr(in.A, in.B, in.C)] = 0
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
		case OpGoto:
			label := labelOf[DecodeAddr(in.A, in.B, in.C)]
			_, _ = fmt.Fprintf(w, "%s\tGoto %d\n", indent, label)
		case OpFunc:
			_, _ = fmt.Fprintf(w, "%s\tFunc %s ", indent, disassembleOperand(fn, in.C, Interface, false))
			disassembleFunction(w, fn.Literals[uint8(in.B)], depth+1)
		default:
			_, _ = fmt.Fprintf(w, "%s\t%s\n", indent, disassembleInstruction(fn, addr))
		}
		switch in.Op {
		case OpCall, OpCallIndirect, OpCallNative, OpTailCall, OpMakeSlice:
			addr += 1
		case OpDefer:
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
	s := operationName[op]
	switch op {
	case OpAddInt, OpAddInt8, OpAddInt16, OpAddInt32,
		OpAnd, OpAndNot, OpOr, OpXor,
		OpDivInt, OpDivInt8, OpDivInt16, OpDivInt32, OpDivUint8, OpDivUint16, OpDivUint32, OpDivUint64,
		OpMulInt, OpMulInt8, OpMulInt16, OpMulInt32,
		OpRemInt, OpRemInt8, OpRemInt16, OpRemInt32, OpRemUint8, OpRemUint16, OpRemUint32, OpRemUint64,
		OpSubInt, OpSubInt8, OpSubInt16, OpSubInt32,
		OpSubInvInt, OpSubInvInt8, OpSubInvInt16, OpSubInvInt32,
		OpLeftShift, OpLeftShift8, OpLeftShift16, OpLeftShift32,
		OpRightShift, OpRightShiftU:
		s += " " + disassembleOperand(fn, a, Int, false)
		s += " " + disassembleOperand(fn, b, Int, k)
		s += " " + disassembleOperand(fn, c, Int, false)
	case OpAddFloat32, OpAddFloat64, OpDivFloat32, OpDivFloat64,
		OpMulFloat32, OpMulFloat64,
		OpSubFloat32, OpSubFloat64, OpSubInvFloat32, OpSubInvFloat64:
		s += " " + disassembleOperand(fn, a, Float64, false)
		s += " " + disassembleOperand(fn, b, Float64, k)
		s += " " + disassembleOperand(fn, c, Float64, false)
	case OpAssert:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " type(" + fn.Types[b].String() + ")"
		t := fn.Types[int(uint(b))]
		var kind = reflectToRegisterKind(t.Kind())
		s += " " + disassembleOperand(fn, c, kind, false)
	case OpAssertInt:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " type(int)"
		s += " " + disassembleOperand(fn, c, Int, false)
	case OpAssertFloat64:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " type(float64)"
		s += " " + disassembleOperand(fn, c, Float64, false)
	case OpAssertString:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " type(string)"
		s += " " + disassembleOperand(fn, c, String, false)
	case OpBind:
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
	case OpCall, OpCallIndirect, OpCallNative, OpTailCall, OpDefer:
		if a != CurrentFunction {
			switch op {
			case OpCall, OpTailCall:
				sf := fn.ScrigoFunctions[uint8(a)]
				s += " " + packageName(sf.Pkg) + "." + sf.Name
			case OpCallNative:
				nf := fn.NativeFunctions[uint8(a)]
				s += " " + packageName(nf.Pkg) + "." + nf.Name
			case OpCallIndirect, OpDefer:
				s += " " + disassembleOperand(fn, a, Interface, false)
			}
		}
		if c != NoVariadic && (op == OpCallIndirect || op == OpCallNative || op == OpDefer) {
			s += " ..." + strconv.Itoa(int(c))
		}
		switch op {
		case OpCall, OpCallIndirect, OpCallNative, OpDefer:
			grow := fn.Body[addr+1]
			s += "\t// Stack shift: " + strconv.Itoa(int(grow.Op)) + ", " + strconv.Itoa(int(grow.A)) + ", " +
				strconv.Itoa(int(grow.B)) + ", " + strconv.Itoa(int(grow.C))
		}
		if op == OpDefer {
			args := fn.Body[addr+2]
			s += "; args: " + strconv.Itoa(int(args.Op)) + ", " + strconv.Itoa(int(args.A)) + ", " +
				strconv.Itoa(int(args.B)) + ", " + strconv.Itoa(int(args.C))

		}
	case OpCap:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " " + disassembleOperand(fn, c, Int, false)
	case OpCase:
		switch reflect.SelectDir(a) {
		case reflect.SelectSend:
			s += " send " + disassembleOperand(fn, b, Int, k) + " " + disassembleOperand(fn, c, Interface, false)
		case reflect.SelectRecv:
			s += " recv " + disassembleOperand(fn, b, Int, false) + " " + disassembleOperand(fn, c, Interface, false)
		default:
			s += " default"
		}
	case OpContinue:
		s += " " + disassembleOperand(fn, b, Int, false)
	case OpCopy:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " " + disassembleOperand(fn, b, Interface, false)
		s += " " + disassembleOperand(fn, c, Int, false)
	case OpConcat:
		s += " " + disassembleOperand(fn, a, String, false)
		s += " " + disassembleOperand(fn, b, String, k)
		s += " " + disassembleOperand(fn, c, String, false)
	case OpConvert:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " " + fn.Types[int(uint(b))].String()
		s += " " + disassembleOperand(fn, c, Interface, false)
	case OpConvertInt, OpConvertUint:
		s += " " + disassembleOperand(fn, a, Int, false)
		s += " " + fn.Types[int(uint(b))].String()
		s += " " + disassembleOperand(fn, c, Interface, false)
	case OpConvertFloat:
		s += " " + disassembleOperand(fn, a, Float64, false)
		s += " " + fn.Types[int(uint(b))].String()
		s += " " + disassembleOperand(fn, c, Interface, false)
	case OpConvertString:
		s += " " + disassembleOperand(fn, a, String, false)
		s += " " + fn.Types[int(uint(b))].String()
		s += " " + disassembleOperand(fn, c, Interface, false)
	case OpDelete:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " " + disassembleOperand(fn, b, Interface, false)
	case OpIf:
		if Condition(b) == ConditionOK {
			s += " OK"
		} else {
			s += " " + disassembleOperand(fn, a, Interface, false)
			s += " " + Condition(b).String()
		}
	case OpIfInt, OpIfUint:
		s += " " + disassembleOperand(fn, a, Int, false)
		s += " " + Condition(b).String()
		if Condition(b) >= ConditionEqual {
			s += " " + disassembleOperand(fn, c, Int, k)
		}
	case OpIfFloat:
		s += " " + disassembleOperand(fn, a, Float64, false)
		s += " " + Condition(b).String()
		s += " " + disassembleOperand(fn, c, Float64, k)
	case OpIfString:
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
	case OpFunc:
		s += " func(" + strconv.Itoa(int(uint8(b))) + ")"
		s += " " + disassembleOperand(fn, c, Int, false)
	case OpGetFunc:
		if a == 0 {
			f := fn.ScrigoFunctions[uint8(b)]
			s += " " + packageName(f.Pkg) + "." + f.Name
		} else {
			f := fn.NativeFunctions[uint8(b)]
			s += " " + packageName(f.Pkg) + "." + f.Name
		}
		s += " " + disassembleOperand(fn, c, Interface, false)
	case OpGetVar:
		s += " " + packageName(fn.Pkg) + "."
		name := fn.Variables[uint8(b)].Name
		if name == "" {
			s += strconv.Itoa(int(uint8(b)))
		} else {
			s += name
		}
		s += " " + disassembleOperand(fn, c, Interface, false)
	case OpGo, OpReturn:
	case OpGoto:
		s += " " + strconv.Itoa(int(DecodeAddr(a, b, c)))
	case OpIndex:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " " + disassembleOperand(fn, b, Int, k)
		s += " " + disassembleOperand(fn, c, Interface, false)
	case OpLen:
		s += " " + strconv.Itoa(int(a))
		if a == 0 {
			s += " " + disassembleOperand(fn, b, String, false)
		} else {
			s += " " + disassembleOperand(fn, b, Interface, false)
		}
		s += " " + disassembleOperand(fn, c, Int, false)
	case OpLoadNumber:
		if a == 0 {
			s += " int"
			s += " " + fmt.Sprintf("%d", fn.Constants.Int[uint8(b)])
			s += " " + disassembleOperand(fn, c, Int, false)
		} else {
			s += " float"
			s += " " + fmt.Sprintf("%f", fn.Constants.Float[uint8(b)])
			s += " " + disassembleOperand(fn, c, Float64, false)
		}

	case OpMakeChan:
		s += " " + fn.Types[int(uint(a))].String()
		s += " " + disassembleOperand(fn, b, Int, k)
		s += " " + disassembleOperand(fn, c, Interface, false)
	case OpMakeMap:
		s += " " + fn.Types[int(uint(a))].String()
		s += " " + disassembleOperand(fn, b, Int, k)
		s += " " + disassembleOperand(fn, c, Interface, false)
	case OpMapIndex:
		//s += " " + disassembleOperand(scrigo, a, Interface, false)
		//key := reflectToRegisterKind()
		//s += " " + disassembleOperand(scrigo, b, Interface, false)
		//s += " " + disassembleOperand(scrigo, c, Interface, false)
	case OpMove:
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
	case OpNew:
		s += " " + fn.Types[int(uint(b))].String()
		s += " " + disassembleOperand(fn, c, Interface, false)
	case OpPanic, OpPrint:
		s += " " + disassembleOperand(fn, a, Interface, false)
	case OpRange:
		//s += " " + disassembleOperand(scrigo, c, Interface, false)
	case OpRangeString:
		s += " " + disassembleOperand(fn, a, Int, false)
		s += " " + disassembleOperand(fn, b, Int, false)
		s += " " + disassembleOperand(fn, c, Interface, k)
	case OpReceive:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " " + disassembleOperand(fn, b, Bool, false)
		s += " " + disassembleOperand(fn, c, Interface, false)
	case OpRecover:
		if c != 0 {
			s += " " + disassembleOperand(fn, c, Interface, false)
		}
	case OpSelector:
		//s += " " + disassembleOperand(scrigo, c, Interface, false)
	case OpSend:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " " + disassembleOperand(fn, c, Interface, false)
	case OpMakeSlice:
		s += " " + fn.Types[int(uint(a))].String()
		s += " " + disassembleOperand(fn, c, Int, false)
		s += " // len: "
		s += fmt.Sprintf("%d", fn.Body[addr+1].A)
		s += ", cap: "
		s += fmt.Sprintf("%d", fn.Body[addr+1].B)
	case OpSetMap:
		s += " " + disassembleOperand(fn, a, Interface, false)
		if k {
			s += fmt.Sprintf(" K(%v)", b)
		} else {
			s += " " + disassembleOperand(fn, b, Interface, false)
		}
		s += " " + disassembleOperand(fn, c, Interface, false)
	case OpSetSlice:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " " + disassembleOperand(fn, b, Int, k)
		s += " " + disassembleOperand(fn, c, Int, false)
	case OpSetVar:
		s += " " + disassembleOperand(fn, b, Interface, false)
		v := fn.Variables[uint8(c)]
		s += " " + packageName(v.Name) + "."
		if v.Name == "" {
			s += strconv.Itoa(int(uint8(c)))
		} else {
			s += v.Name
		}
	case OpSliceIndex:
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

var operationName = [...]string{

	OpNone: "Nop", // TODO(Gianluca): review.

	OpAddInt:     "Add",
	OpAddInt8:    "Add8",
	OpAddInt16:   "Add16",
	OpAddInt32:   "Add32",
	OpAddFloat32: "Add32",
	OpAddFloat64: "Add",

	OpAnd: "And",

	OpAndNot: "AndNot",

	OpAppend: "Append",

	OpAppendSlice: "AppendSlice",

	OpAssert:        "Assert",
	OpAssertInt:     "Assert",
	OpAssertFloat64: "Assert",
	OpAssertString:  "Assert",

	OpBind: "Bind",

	OpCall: "Call",

	OpCallIndirect: "CallIndirect",

	OpCallNative: "CallNative",

	OpCap: "Cap",

	OpCase: "Case",

	OpContinue: "Continue",

	OpConvert:       "Convert",
	OpConvertInt:    "Convert",
	OpConvertUint:   "ConvertU",
	OpConvertFloat:  "Convert",
	OpConvertString: "Convert",

	OpCopy: "Copy",

	OpConcat: "concat",

	OpDefer: "Defer",

	OpDelete: "delete",

	OpDivInt:     "Div",
	OpDivInt8:    "Div8",
	OpDivInt16:   "Div16",
	OpDivInt32:   "Div32",
	OpDivUint8:   "DivU8",
	OpDivUint16:  "DivU16",
	OpDivUint32:  "DivU32",
	OpDivUint64:  "DivU64",
	OpDivFloat32: "Div32",
	OpDivFloat64: "Div",

	OpFunc: "Func",

	OpGetFunc: "GetFunc",

	OpGetVar: "GetVar",

	OpGo: "Go",

	OpGoto: "Goto",

	OpIf:       "If",
	OpIfInt:    "If",
	OpIfUint:   "IfU",
	OpIfFloat:  "If",
	OpIfString: "If",

	OpIndex: "Index",

	OpLeftShift:   "LeftShift",
	OpLeftShift8:  "LeftShift8",
	OpLeftShift16: "LeftShift16",
	OpLeftShift32: "LeftShift32",

	OpLen: "Len",

	OpLoadNumber: "LoadNumber",

	OpMakeChan: "MakeChan",

	OpMapIndex: "MapIndex",

	OpMakeMap: "MakeMap",

	OpMakeSlice: "MakeSlice",

	OpMove: "Move",

	OpMulInt:     "Mul",
	OpMulInt8:    "Mul8",
	OpMulInt16:   "Mul16",
	OpMulInt32:   "Mul32",
	OpMulFloat32: "Mul32",
	OpMulFloat64: "Mul",

	OpNew: "New",

	OpOr: "Or",

	OpPanic: "Panic",

	OpPrint: "Print",

	OpRange: "Range",

	OpRangeString: "Range",

	OpReceive: "Receive",

	OpRecover: "Recover",

	OpRemInt:    "Rem",
	OpRemInt8:   "Rem8",
	OpRemInt16:  "Rem16",
	OpRemInt32:  "Rem32",
	OpRemUint8:  "RemU8",
	OpRemUint16: "RemU16",
	OpRemUint32: "RemU32",
	OpRemUint64: "RemU64",

	OpReturn: "Return",

	OpRightShift:  "RightShift",
	OpRightShiftU: "RightShiftU",

	OpSelect: "Select",

	OpSelector: "Selector",

	OpSend: "Send",

	OpSetMap: "SetMap",

	OpSetSlice: "SetSlice",

	OpSetVar: "SetPackageVar",

	OpSliceIndex: "SliceIndex",

	OpStringIndex: "StringIndex",

	OpSubInt:     "Sub",
	OpSubInt8:    "Sub8",
	OpSubInt16:   "Sub16",
	OpSubInt32:   "Sub32",
	OpSubFloat32: "Sub32",
	OpSubFloat64: "Sub",

	OpSubInvInt:     "SubInv",
	OpSubInvInt8:    "SubInv8",
	OpSubInvInt16:   "SubInv16",
	OpSubInvInt32:   "SubInv32",
	OpSubInvFloat32: "SubInv32",
	OpSubInvFloat64: "SubInv",

	OpTailCall: "TailCall",

	OpXor: "Xor",
}
