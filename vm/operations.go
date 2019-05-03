// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vm

type operation int8

const (
	opNone operation = iota

	opAddInt
	opAddInt8
	opAddInt16
	opAddInt32
	opAddFloat32
	opAddFloat64

	opAnd

	opAndNot

	opAssert
	opAssertInt
	opAssertFloat64
	opAssertString

	opAppend

	opAppendSlice

	opBind

	opCall

	opCallIndirect

	opCallNative

	opCap

	opContinue

	opConvert
	opConvertInt
	opConvertUint
	opConvertFloat
	opConvertString

	opCopy

	opConcat

	opDefer

	opDelete

	opDivInt
	opDivInt8
	opDivInt16
	opDivInt32
	opDivUint8
	opDivUint16
	opDivUint32
	opDivUint64
	opDivFloat32
	opDivFloat64

	opFunc

	opGetFunc

	opGetVar

	opGo

	opGoto

	opIf
	opIfInt
	opIfUint
	opIfFloat
	opIfString

	opIndex

	opLen

	opMakeChan

	opMakeMap

	opMakeSlice

	opMapIndex

	opMapIndexStringBool

	opMapIndexStringInt

	opMapIndexStringInterface

	opMapIndexStringString

	opMove

	opMulInt
	opMulInt8
	opMulInt16
	opMulInt32
	opMulFloat32
	opMulFloat64

	opNew

	opOr

	opPanic

	opPrint

	opRange

	opRangeString

	opReceive

	opRecover

	opRemInt
	opRemInt8
	opRemInt16
	opRemInt32
	opRemUint8
	opRemUint16
	opRemUint32
	opRemUint64

	opReturn

	opSelector

	opSend

	opSetMap

	opSetSlice

	opSetVar

	opSliceIndex

	opStringIndex

	opSubInt
	opSubInt8
	opSubInt16
	opSubInt32
	opSubFloat32
	opSubFloat64

	opSubInvInt
	opSubInvInt8
	opSubInvInt16
	opSubInvInt32
	opSubInvFloat32
	opSubInvFloat64

	opTailCall

	opXor
)

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

	opLen: "Len",

	opMakeChan: "MakeChan",

	opMakeMap: "MakeMap",

	opMakeSlice: "MakeSlice",

	opMapIndex: "MapIndex",

	opMapIndexStringBool: "MapIndexStringBool",

	opMapIndexStringInt: "MapIndexStringInt",

	opMapIndexStringInterface: "MapIndexStringInterface",

	opMapIndexStringString: "MapIndexStringString",

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
}
