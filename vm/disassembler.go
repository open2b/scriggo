// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

func Disassemble(w io.Writer, pkg *Package) (int64, error) {
	var b bytes.Buffer
	_, _ = fmt.Fprintf(&b, "\nPackage %s\n", pkg.name)
	//if len(main.consts.Int) > 0 {
	//	_, _ = fmt.Fprintf(&b, "\nconst int\n")
	//	for i, value := range main.consts.Int {
	//		_, _ = fmt.Fprintf(&b, "\t%d\t%#v\n", i, value)
	//	}
	//}
	//if len(main.consts.Float) > 0 {
	//	_, _ = fmt.Fprintf(&b, "\nconst float\n")
	//	for i, value := range main.consts.Float {
	//		_, _ = fmt.Fprintf(&b, "\t%d\t%#v\n", i, value)
	//	}
	//}
	//if len(main.consts.String) > 0 {
	//	_, _ = fmt.Fprintf(&b, "\nconst string\n")
	//	for i, value := range main.consts.String {
	//		_, _ = fmt.Fprintf(&b, "\t%d\t%#v\n", i, value)
	//	}
	//}
	//if len(main.consts.Iface) > 0 {
	//	_, _ = fmt.Fprintf(&b, "\nconst\n")
	//	for i, value := range main.consts.Iface {
	//		_, _ = fmt.Fprintf(&b, "\t%d\t%#v\n", i, value)
	//	}
	//}
	_, _ = fmt.Fprint(&b, "\n")
	if len(pkg.packages) > 0 {
		for _, p := range pkg.packages {
			_, _ = fmt.Fprintf(&b, "Import %q\n", p.name)
		}
		_, _ = fmt.Fprint(&b, "\n")
	}
	for i, fn := range pkg.scrigoFunctions {
		if i > 0 {
			_, _ = b.WriteString("\n")
		}
		_, _ = fmt.Fprintf(&b, "Func %s", fn.name)
		disassembleFunction(&b, fn, 0)
	}
	_, _ = fmt.Fprint(&b, "\n")
	return b.WriteTo(w)
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
		case opCall, opCallFunc, opCallMethod, opTailCall:
			addr += 1
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
		s += " " + disassembleOperand(fn, c, Int, false)
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
	case opCall, opCallFunc, opCallMethod, opTailCall:
		if a == NoPackage {
			s += " " + disassembleOperand(fn, b, Interface, false)
		} else {
			if op == opCallMethod {
				t := fn.types[int(uint8(a))]
				m := t.Method(int(uint8(b)))
				s += " " + t.String() + "." + m.Name
			} else if b != CurrentFunction {
				pkg := fn.pkg
				if a != CurrentPackage {
					pkg = pkg.packages[uint8(a)]
				}
				s += " " + pkg.name + "."
				switch op {
				case opCall, opTailCall:
					s += pkg.scrigoFunctions[uint8(b)].name
				case opCallFunc:
					name := pkg.nativeFunctions[uint8(b)].name
					if name == "" {
						s += strconv.Itoa(int(uint8(b)))
					} else {
						s += name
					}
				}
			}
		}
		if c != NoVariadic && (op == opCallFunc || op == opCallMethod) {
			s += " ..." + strconv.Itoa(int(c))
		}
		switch op {
		case opCall, opCallFunc, opCallMethod:
			grow := fn.body[addr+1]
			s += "\t// Stack shift: " + strconv.Itoa(int(grow.op)) + ", " + strconv.Itoa(int(grow.a)) + ", " +
				strconv.Itoa(int(grow.b)) + ", " + strconv.Itoa(int(grow.c))
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
	case opDelete:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " " + disassembleOperand(fn, b, Interface, false)
	case opIf:
		s += " " + disassembleOperand(fn, a, Interface, false)
		s += " " + Condition(b).String()
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
		pkg := fn.pkg
		if a != CurrentPackage {
			pkg = pkg.packages[uint8(a)]
		}
		s += " " + pkg.name + "." + pkg.scrigoFunctions[uint8(b)].name
		s += " " + disassembleOperand(fn, c, Interface, false)
	case opGetVar:
		pkg := fn.pkg
		if a != CurrentPackage {
			pkg = pkg.packages[uint8(a)]
		}
		s += " " + pkg.name + "."
		if pkg.varNames == nil {
			s += strconv.Itoa(int(uint8(b)))
		} else {
			s += pkg.varNames[uint8(b)]
		}
		s += " " + disassembleOperand(fn, c, Interface, false)
	case opGo, opReturn:
	case opGoto, opJmpOk, opJmpNotOk:
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
	case opMakeMap:
		s += " type(" + strconv.Itoa(int(uint(a))) + ")"
		s += " " + disassembleOperand(fn, b, Int, false)
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
		s += " " + disassembleOperand(fn, b, Interface, k)
		s += " " + disassembleOperand(fn, c, Interface, false)
	case opMoveInt:
		s += " " + disassembleOperand(fn, b, Int, k)
		s += " " + disassembleOperand(fn, c, Int, false)
	case opMoveFloat:
		s += " " + disassembleOperand(fn, b, Float64, k)
		s += " " + disassembleOperand(fn, c, Float64, false)
	case opMoveString:
		s += " " + disassembleOperand(fn, b, String, k)
		s += " " + disassembleOperand(fn, c, String, false)
	case opNew:
		s += " " + fn.types[int(uint(b))].String()
		s += " " + disassembleOperand(fn, c, Interface, false)
	case opRange:
		//s += " " + disassembleOperand(scrigo, c, Interface, false)
	case opRangeString:
		s += " " + disassembleOperand(fn, a, Int, false)
		s += " " + disassembleOperand(fn, b, Int, false)
		s += " " + disassembleOperand(fn, c, Interface, k)
	case opSelector:
		//s += " " + disassembleOperand(scrigo, c, Interface, false)
	case opMakeSlice:
		//s += " " + disassembleOperand(scrigo, c, Interface, false)
	case opSetVar:
		s += " " + disassembleOperand(fn, a, Interface, false)
		pkg := fn.pkg
		if b != CurrentPackage {
			pkg = pkg.packages[uint8(b)]
		}
		s += " " + pkg.name + "."
		if pkg.varNames == nil {
			s += strconv.Itoa(int(uint8(c)))
		} else {
			s += pkg.varNames[uint8(c)]
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
			return fmt.Sprintf("%q", fn.constants.General[uint8(op)])
		}
	}
	if op > 0 {
		return "R" + strconv.Itoa(int(op))
	}
	if op == 0 {
		return "NR"
	}
	return "(R" + strconv.Itoa(-int(op)) + ")"
}
