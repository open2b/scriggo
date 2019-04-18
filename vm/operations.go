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

	opBind

	opCall

	opCallIndirect

	opCallFunc

	opCallMethod

	opCap

	opContinue

	opConvert
	opConvertInt
	opConvertUint
	opConvertFloat
	opConvertString

	opCopy

	opConcat

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

	opStringIndex

	opJmpOk
	opJmpNotOk

	opLen

	opFunc

	opMakeChan

	opMakeMap

	opMapIndex
	opMapIndexStringInt
	opMapIndexStringBool
	opMapIndexStringString
	opMapIndexStringInterface

	opMove
	opMoveInt
	opMoveFloat
	opMoveString

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

	opSetVar

	opMakeSlice

	opSetSlice
	opSetSliceInt
	opSetSliceFloat
	opSetSliceString

	opSend

	opSliceIndex

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

	opAddInt:     "AddInt",
	opAddInt8:    "AddInt8",
	opAddInt16:   "AddInt16",
	opAddInt32:   "AddInt32",
	opAddFloat32: "AddFloat32",
	opAddFloat64: "AddFloat64",

	opAnd: "And",

	opAndNot: "AndNot",

	opAppend: "Append",

	opAssert:        "Assert",
	opAssertInt:     "AssertInt",
	opAssertFloat64: "AssertFloat64",
	opAssertString:  "AssertString",

	opBind: "Bind",

	opCall: "Call",

	opCallFunc: "CallFunc",

	opCallIndirect: "CallIndirect",

	opCallMethod: "CallMethod",

	opCap: "Cap",

	opCopy: "Copy",

	opConcat: "concat",

	opConvert:       "Convert",
	opConvertInt:    "ConvertInt",
	opConvertUint:   "ConvertUint",
	opConvertFloat:  "ConvertFloat",
	opConvertString: "ConvertString",

	opDelete: "delete",

	opDivInt:     "DivInt",
	opDivInt8:    "DivInt8",
	opDivInt16:   "DivInt16",
	opDivInt32:   "DivInt32",
	opDivUint8:   "DivUint8",
	opDivUint16:  "DivUint16",
	opDivUint32:  "DivUint32",
	opDivUint64:  "DivUint64",
	opDivFloat32: "DivFloat32",
	opDivFloat64: "DivFloat64",

	opGetFunc: "GetFunc",

	opGetVar: "GetVar",

	opGo: "Go",

	opGoto: "Goto",

	opIf:       "If",
	opIfInt:    "IfInt",
	opIfUint:   "IfUint",
	opIfFloat:  "IfFloat",
	opIfString: "IfString",

	opJmpOk:    "JmpOk",
	opJmpNotOk: "JmpNotOk",

	opLen: "len",

	opFunc: "Func",

	opMakeChan: "MakeChan",

	opMakeMap: "MakeMap",

	opMove:       "Move",
	opMoveInt:    "MoveInt",
	opMoveFloat:  "MoveFloat",
	opMoveString: "MoveString",

	opMulInt:     "MulInt",
	opMulInt8:    "MulInt8",
	opMulInt16:   "MulInt16",
	opMulInt32:   "MulInt32",
	opMulFloat32: "MulFloat32",
	opMulFloat64: "MulFloat64",

	opNew: "New",

	opRange:       "Range",
	opRangeString: "RangeString",

	opPanic: "Panic",

	opPrint: "Print",

	opReceive: "Receive",

	opRemInt:    "RemInt",
	opRemInt8:   "RemInt8",
	opRemInt16:  "RemInt16",
	opRemInt32:  "RemInt32",
	opRemUint8:  "RemUint8",
	opRemUint16: "RemUint16",
	opRemUint32: "RemUint32",
	opRemUint64: "RemUint64",

	opReturn: "Return",

	opMakeSlice: "MakeSlice",

	opSetSlice:       "SetSlice",
	opSetSliceInt:    "SetSliceInt",
	opSetSliceFloat:  "SetSliceFloat",
	opSetSliceString: "SetSliceString",

	opSend: "Send",

	opSetVar: "SetPackageVar",

	opSubInt:     "SubInt",
	opSubInt8:    "SubInt8",
	opSubInt16:   "SubInt16",
	opSubInt32:   "SubInt32",
	opSubFloat32: "SubFloat32",
	opSubFloat64: "SubFloat64",

	opSubInvInt:     "SubInvInt",
	opSubInvInt8:    "SubInvInt8",
	opSubInvInt16:   "SubInvInt16",
	opSubInvInt32:   "SubInvInt32",
	opSubInvFloat32: "SubInvFloat32",
	opSubInvFloat64: "SubInvFloat64",

	opTailCall: "TailCall",
}
