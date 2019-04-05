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

func disassembleFunction(b *bytes.Buffer, fn *Function) {
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
	_, _ = fmt.Fprintf(b, "Func %s(", fn.name)
	var reg uint8
	for i, typ := range fn.in {
		if i > 0 {
			_, _ = fmt.Fprint(b, ", ")
		}
		_, _ = fmt.Fprintf(b, "%d %s", reg, typ)

		reg++
	}
	_, _ = fmt.Fprint(b, ")")
	if len(fn.out) > 0 {
		_, _ = fmt.Fprint(b, " (")
		for i, typ := range fn.out {
			if i > 0 {
				_, _ = fmt.Fprint(b, ", ")
			}
			_, _ = fmt.Fprintf(b, "%d %s", reg, typ)
			reg++
		}
		_, _ = fmt.Fprint(b, ")")
	}
	_, _ = fmt.Fprint(b, "\n")
	_, _ = fmt.Fprintf(b, "\t// regs(%d,%d,%d,%d) in(%d,%d,%d,%d) out(%d,%d,%d,%d)\n",
		fn.numRegs[0], fn.numRegs[1], fn.numRegs[2], fn.numRegs[3],
		fn.numIn[0], fn.numIn[1], fn.numIn[2], fn.numIn[3],
		fn.numOut[0], fn.numOut[1], fn.numOut[2], fn.numOut[3])
	for addr, in := range fn.body {
		if label, ok := labelOf[uint32(addr)]; ok {
			_, _ = fmt.Fprintf(b, "%d:", label)
		}
		if in.op == opGoto {
			label := labelOf[decodeAddr(in.a, in.b, in.c)]
			_, _ = fmt.Fprintf(b, "\tGoto %d\n", label)
		} else {
			_, _ = fmt.Fprintf(b, "\t%s\n", disassembleInstruction(fn, in))
		}
	}
}

func DisassembleInstruction(w io.Writer, fn *Function, in instruction) (int64, error) {
	n, err := io.WriteString(w, disassembleInstruction(fn, in))
	return int64(n), err
}

func disassembleInstruction(fn *Function, in instruction) string {
	s := in.op.String()
	switch in.op {
	case opAddInt, opAddInt8, opAddInt16, opAddInt32,
		opAnd, opAndNot, opOr, opXor,
		opDivInt, opDivInt8, opDivInt16, opDivInt32, opDivUint8, opDivUint16, opDivUint32, opDivUint64,
		opMulInt, opMulInt8, opMulInt16, opMulInt32,
		opRemInt, opRemInt8, opRemInt16, opRemInt32, opRemUint8, opRemUint16, opRemUint32, opRemUint64,
		opSubInt, opSubInt8, opSubInt16, opSubInt32,
		opSubInvInt, opSubInvInt8, opSubInvInt16, opSubInvInt32:
		s += " " + disassembleOperand(in.a, Int, false)
		s += " " + disassembleOperand(in.b, Int, in.k)
		s += " " + disassembleOperand(in.c, Int, false)
	case opAddFloat32, opAddFloat64, opDivFloat32, opDivFloat64,
		opMulFloat32, opMulFloat64,
		opSubFloat32, opSubFloat64, opSubInvFloat32, opSubInvFloat64:
		s += " " + disassembleOperand(in.a, Float64, false)
		s += " " + disassembleOperand(in.b, Float64, in.k)
		s += " " + disassembleOperand(in.c, Float64, false)
	case opAssert:
		s += " " + disassembleOperand(in.a, Interface, false)
		s += " type(" + strconv.Itoa(int(uint(in.b))) + ")"
		t := fn.types[int(uint(in.b))]
		var kind = reflectToRegisterKind(t.Kind())
		s += " " + disassembleOperand(in.c, kind, false)
	case opAssertInt:
		s += " " + disassembleOperand(in.a, Interface, false)
		s += " type(int)"
		s += " " + disassembleOperand(in.c, Int, false)
	case opAssertFloat64:
		s += " " + disassembleOperand(in.a, Interface, false)
		s += " type(float64)"
		s += " " + disassembleOperand(in.c, Int, false)
	case opAssertString:
		s += " " + disassembleOperand(in.a, Interface, false)
		s += " type(string)"
		s += " " + disassembleOperand(in.c, String, false)
	case opCall:
		s += " " + disassembleOperand(in.a, Interface, false)
	case opCap:
		s += " " + disassembleOperand(in.a, Interface, false)
		s += " " + disassembleOperand(in.c, Int, false)
	case opContinue:
		s += " " + disassembleOperand(in.b, Int, false)
	case opCopy:
		s += " " + disassembleOperand(in.a, Interface, false)
		s += " " + disassembleOperand(in.b, Interface, false)
		s += " " + disassembleOperand(in.c, Int, false)
	case opConcat:
		s += " " + disassembleOperand(in.a, String, false)
		s += " " + disassembleOperand(in.b, String, in.k)
		s += " " + disassembleOperand(in.c, String, false)
	case opDelete:
		s += " " + disassembleOperand(in.a, Interface, false)
		s += " " + disassembleOperand(in.b, Interface, false)
	case opIf:
		s += " " + disassembleOperand(in.a, Interface, false)
		s += " " + Condition(in.b).String()
	case opIfInt, opIfUint:
		s += " " + disassembleOperand(in.a, Int, false)
		s += " " + Condition(in.b).String()
		if Condition(in.b) >= ConditionEqual {
			s += " " + disassembleOperand(in.c, Int, in.k)
		}
	case opIfFloat:
		s += " " + disassembleOperand(in.a, Float64, false)
		s += " " + Condition(in.b).String()
		s += " " + disassembleOperand(in.c, Float64, in.k)
	case opIfString:
		s += " " + disassembleOperand(in.a, String, false)
		s += " " + Condition(in.b).String()
		if Condition(in.b) < ConditionEqualLen {
			if in.k && in.c >= 0 {
				s += " " + strconv.Quote(string(in.c))
			} else {
				s += " " + disassembleOperand(in.c, String, in.k)
			}
		} else {
			s += " " + disassembleOperand(in.c, Int, in.k)
		}
	case opGoto, opJmpOk, opJmpNotOk:
		s += strconv.Itoa(int(decodeAddr(in.a, in.b, in.c)))
	case opIndex:
		s += " " + disassembleOperand(in.a, Interface, false)
		s += " " + disassembleOperand(in.b, Int, in.k)
		s += " " + disassembleOperand(in.c, Interface, false)
	case opLen:
		s += " " + strconv.Itoa(int(in.a))
		if in.a == 0 {
			s += " " + disassembleOperand(in.b, String, false)
		} else {
			s += " " + disassembleOperand(in.b, Interface, false)
		}
		s += " " + disassembleOperand(in.c, Int, false)
	case opMakeMap:
		s += " type(" + strconv.Itoa(int(uint(in.a))) + ")"
		s += " " + disassembleOperand(in.b, Int, false)
		s += " " + disassembleOperand(in.c, Interface, false)
	case opMapIndex:
		//s += " " + disassembleOperand(in.a, Interface, false)
		//key := reflectToRegisterKind()
		//s += " " + disassembleOperand(in.b, Interface, false)
		//s += " " + disassembleOperand(in.c, Interface, false)
	case opMapIndexStringString:
		s += " " + disassembleOperand(in.a, Interface, false)
		s += " " + disassembleOperand(in.b, String, in.k)
		s += " " + disassembleOperand(in.c, String, false)
	case opMapIndexStringInt:
		s += " " + disassembleOperand(in.a, Interface, false)
		s += " " + disassembleOperand(in.b, String, in.k)
		s += " " + disassembleOperand(in.c, String, false)
	case opMapIndexStringInterface:
		s += " " + disassembleOperand(in.a, Interface, false)
		s += " " + disassembleOperand(in.b, String, in.k)
		s += " " + disassembleOperand(in.c, Interface, false)
	case opMove:
		s += " " + disassembleOperand(in.a, Interface, false)
		s += " " + disassembleOperand(in.c, Interface, false)
	case opMoveInt:
		s += " " + disassembleOperand(in.b, Int, in.k)
		s += " " + disassembleOperand(in.c, Int, false)
	case opMoveFloat:
		s += " " + disassembleOperand(in.b, Float64, in.k)
		s += " " + disassembleOperand(in.c, Float64, false)
	case opMoveString:
		s += " " + disassembleOperand(in.b, String, in.k)
		s += " " + disassembleOperand(in.c, String, false)
	case opNew:
		s += " type(" + strconv.Itoa(int(uint(in.b))) + ")"
		s += " " + disassembleOperand(in.c, Interface, false)
	case opRange:
		//s += " " + disassembleOperand(in.c, Interface, false)
	case opRangeString:
		s += " " + disassembleOperand(in.a, Int, false)
		s += " " + disassembleOperand(in.b, Int, false)
		s += " " + disassembleOperand(in.c, Interface, in.k)
	case opReturn:
	case opSelector:
		//s += " " + disassembleOperand(in.c, Interface, false)
	case opMakeSlice:
		//s += " " + disassembleOperand(in.c, Interface, false)
	case opSliceIndex:
		s += " " + disassembleOperand(in.a, Interface, false)
		s += " " + disassembleOperand(in.b, Int, in.k)
		//s += " " + disassembleOperand(in.c, Interface, false)
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
	return "G" + strconv.Itoa(-int(op))
}
