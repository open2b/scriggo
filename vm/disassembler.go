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
)

func Disassemble(w io.Writer, pkg *Package) (int64, error) {
	var b bytes.Buffer
	_, _ = fmt.Fprintf(&b, "Package %s\n", pkg.name)
	//if len(pkg.consts.Int) > 0 {
	//	_, _ = fmt.Fprintf(&b, "\nconst int\n")
	//	for i, value := range pkg.consts.Int {
	//		_, _ = fmt.Fprintf(&b, "\t%d\t%#v\n", i, value)
	//	}
	//}
	//if len(pkg.consts.Float) > 0 {
	//	_, _ = fmt.Fprintf(&b, "\nconst float\n")
	//	for i, value := range pkg.consts.Float {
	//		_, _ = fmt.Fprintf(&b, "\t%d\t%#v\n", i, value)
	//	}
	//}
	//if len(pkg.consts.String) > 0 {
	//	_, _ = fmt.Fprintf(&b, "\nconst string\n")
	//	for i, value := range pkg.consts.String {
	//		_, _ = fmt.Fprintf(&b, "\t%d\t%#v\n", i, value)
	//	}
	//}
	//if len(pkg.consts.Iface) > 0 {
	//	_, _ = fmt.Fprintf(&b, "\nconst\n")
	//	for i, value := range pkg.consts.Iface {
	//		_, _ = fmt.Fprintf(&b, "\t%d\t%#v\n", i, value)
	//	}
	//}
	_, _ = fmt.Fprint(&b, "\n")
	for i, fn := range pkg.funcs {
		if i > 0 {
			_, _ = b.WriteString("\n")
		}
		disassembleFunction(&b, fn)
	}
	return b.WriteTo(w)
}

func DisassembleFunction(w io.Writer, fn *Function) (int64, error) {
	var b bytes.Buffer
	disassembleFunction(&b, fn)
	return b.WriteTo(w)
}

func disassembleFunction(w *bytes.Buffer, fn *Function) {
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
	_, _ = fmt.Fprintf(w, "Func %s(", fn.name)
	for i, typ := range fn.in {
		if i > 0 {
			_, _ = fmt.Fprint(w, ", ")
		}
		_, _ = fmt.Fprintf(w, "%d %s", len(fn.out)+i, typ)
	}
	_, _ = fmt.Fprint(w, ")")
	if len(fn.out) > 0 {
		_, _ = fmt.Fprint(w, " (")
		for i, typ := range fn.out {
			if i > 0 {
				_, _ = fmt.Fprint(w, ", ")
			}
			_, _ = fmt.Fprintf(w, "%d %s", i, typ)
		}
		_, _ = fmt.Fprint(w, ")")
	}
	_, _ = fmt.Fprint(w, "\n")
	_, _ = fmt.Fprintf(w, "\t// regs(%d,%d,%d,%d)\n",
		fn.regnum[0], fn.regnum[1], fn.regnum[2], fn.regnum[3])
	instrNum := uint32(len(fn.body))
	for addr := uint32(0); addr < instrNum; addr++ {
		if label, ok := labelOf[uint32(addr)]; ok {
			_, _ = fmt.Fprintf(w, "%d:", label)
		}
		in := fn.body[addr]
		if in.op == opGoto {
			label := labelOf[decodeAddr(in.a, in.b, in.c)]
			_, _ = fmt.Fprintf(w, "\tGoto %d\n", label)
		} else {
			_, _ = fmt.Fprintf(w, "\t%s\n", disassembleInstruction(fn, addr))
		}
		if in.op == opCall {
			addr += 1
		}
	}
}

func DisassembleInstruction(w io.Writer, fn *Function, addr uint32) (int64, error) {
	n, err := io.WriteString(w, disassembleInstruction(fn, addr))
	return int64(n), err
}

func disassembleInstruction(fn *Function, addr uint32) string {
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
		s += " " + disassembleOperand(a, Int, false)
		s += " " + disassembleOperand(b, Int, k)
		s += " " + disassembleOperand(c, Int, false)
	case opAddFloat32, opAddFloat64, opDivFloat32, opDivFloat64,
		opMulFloat32, opMulFloat64,
		opSubFloat32, opSubFloat64, opSubInvFloat32, opSubInvFloat64:
		s += " " + disassembleOperand(a, Float64, false)
		s += " " + disassembleOperand(b, Float64, k)
		s += " " + disassembleOperand(c, Float64, false)
	case opAssert:
		s += " " + disassembleOperand(a, Interface, false)
		s += " type(" + strconv.Itoa(int(uint(b))) + ")"
		t := fn.types[int(uint(b))]
		var kind = reflectToRegisterKind(t.Kind())
		s += " " + disassembleOperand(c, kind, false)
	case opAssertInt:
		s += " " + disassembleOperand(a, Interface, false)
		s += " type(int)"
		s += " " + disassembleOperand(c, Int, false)
	case opAssertFloat64:
		s += " " + disassembleOperand(a, Interface, false)
		s += " type(float64)"
		s += " " + disassembleOperand(c, Int, false)
	case opAssertString:
		s += " " + disassembleOperand(a, Interface, false)
		s += " type(string)"
		s += " " + disassembleOperand(c, String, false)
	case opCall:
		grow := fn.body[addr+1]
		s += " " + disassembleOperand(a, Interface, false) + " [" +
			strconv.Itoa(int(grow.op)) + "," +
			strconv.Itoa(int(grow.a)) + "," +
			strconv.Itoa(int(grow.b)) + "," +
			strconv.Itoa(int(grow.c)) + "]"
	case opCap:
		s += " " + disassembleOperand(a, Interface, false)
		s += " " + disassembleOperand(c, Int, false)
	case opContinue:
		s += " " + disassembleOperand(b, Int, false)
	case opCopy:
		s += " " + disassembleOperand(a, Interface, false)
		s += " " + disassembleOperand(b, Interface, false)
		s += " " + disassembleOperand(c, Int, false)
	case opConcat:
		s += " " + disassembleOperand(a, String, false)
		s += " " + disassembleOperand(b, String, k)
		s += " " + disassembleOperand(c, String, false)
	case opDelete:
		s += " " + disassembleOperand(a, Interface, false)
		s += " " + disassembleOperand(b, Interface, false)
	case opIf:
		s += " " + disassembleOperand(a, Interface, false)
		s += " " + Condition(b).String()
	case opIfInt, opIfUint:
		s += " " + disassembleOperand(a, Int, false)
		s += " " + Condition(b).String()
		if Condition(b) >= ConditionEqual {
			s += " " + disassembleOperand(c, Int, k)
		}
	case opIfFloat:
		s += " " + disassembleOperand(a, Float64, false)
		s += " " + Condition(b).String()
		s += " " + disassembleOperand(c, Float64, k)
	case opIfString:
		s += " " + disassembleOperand(a, String, false)
		s += " " + Condition(b).String()
		if Condition(b) < ConditionEqualLen {
			if k && c >= 0 {
				s += " " + strconv.Quote(string(c))
			} else {
				s += " " + disassembleOperand(c, String, k)
			}
		} else {
			s += " " + disassembleOperand(c, Int, k)
		}
	case opGoto, opJmpOk, opJmpNotOk:
		s += " " + strconv.Itoa(int(decodeAddr(a, b, c)))
	case opIndex:
		s += " " + disassembleOperand(a, Interface, false)
		s += " " + disassembleOperand(b, Int, k)
		s += " " + disassembleOperand(c, Interface, false)
	case opLen:
		s += " " + strconv.Itoa(int(a))
		if a == 0 {
			s += " " + disassembleOperand(b, String, false)
		} else {
			s += " " + disassembleOperand(b, Interface, false)
		}
		s += " " + disassembleOperand(c, Int, false)
	case opMakeMap:
		s += " type(" + strconv.Itoa(int(uint(a))) + ")"
		s += " " + disassembleOperand(b, Int, false)
		s += " " + disassembleOperand(c, Interface, false)
	case opMapIndex:
		//s += " " + disassembleOperand(a, Interface, false)
		//key := reflectToRegisterKind()
		//s += " " + disassembleOperand(b, Interface, false)
		//s += " " + disassembleOperand(c, Interface, false)
	case opMapIndexStringString:
		s += " " + disassembleOperand(a, Interface, false)
		s += " " + disassembleOperand(b, String, k)
		s += " " + disassembleOperand(c, String, false)
	case opMapIndexStringInt:
		s += " " + disassembleOperand(a, Interface, false)
		s += " " + disassembleOperand(b, String, k)
		s += " " + disassembleOperand(c, String, false)
	case opMapIndexStringInterface:
		s += " " + disassembleOperand(a, Interface, false)
		s += " " + disassembleOperand(b, String, k)
		s += " " + disassembleOperand(c, Interface, false)
	case opMove:
		s += " " + disassembleOperand(a, Interface, false)
		s += " " + disassembleOperand(c, Interface, false)
	case opMoveInt:
		s += " " + disassembleOperand(b, Int, k)
		s += " " + disassembleOperand(c, Int, false)
	case opMoveFloat:
		s += " " + disassembleOperand(b, Float64, k)
		s += " " + disassembleOperand(c, Float64, false)
	case opMoveString:
		s += " " + disassembleOperand(b, String, k)
		s += " " + disassembleOperand(c, String, false)
	case opNew:
		s += " type(" + strconv.Itoa(int(uint(b))) + ")"
		s += " " + disassembleOperand(c, Interface, false)
	case opRange:
		//s += " " + disassembleOperand(c, Interface, false)
	case opRangeString:
		s += " " + disassembleOperand(a, Int, false)
		s += " " + disassembleOperand(b, Int, false)
		s += " " + disassembleOperand(c, Interface, k)
	case opReturn:
	case opSelector:
		//s += " " + disassembleOperand(c, Interface, false)
	case opNewEmptySlice:
		//s += " " + disassembleOperand(c, Interface, false)
	case opSliceIndex:
		s += " " + disassembleOperand(a, Interface, false)
		s += " " + disassembleOperand(b, Int, k)
		//s += " " + disassembleOperand(c, Interface, false)
	case opTailCall:
		s += " " + disassembleOperand(a, Interface, false)
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

func disassembleOperand(op int8, kind Kind, constant bool) string {
	if constant {
		if op >= 0 {
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
				return strconv.Quote(string(op))
			}
		}
		return "K" + strconv.Itoa(-int(op))
	}
	if op >= 0 {
		return "R" + strconv.Itoa(int(op))
	}
	if op == NoRegister {
		return "NR"
	}
	return "G" + strconv.Itoa(-int(op))
}
