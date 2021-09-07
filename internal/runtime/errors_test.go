// Copyright 2021 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"strings"
	"testing"
)

type myBool bool
type myInt int
type myInt8 int8
type myInt16 int16
type myInt32 int32
type myInt64 int64
type myUint uint
type myUint8 uint8
type myUint16 uint16
type myUint32 uint32
type myUint64 uint64
type myUintptr uintptr
type myFloat32 float32
type myFloat64 float64
type myComplex64 complex64
type myComplex128 complex128
type myString string
type myStruct struct{}

var tests = []struct {
	val interface{}
	str string
}{
	{myBool(true), "myBool(true)"},
	{myInt(73), "myInt(73)"},
	{myInt8(73), "myInt8(73)"},
	{myInt16(73), "myInt16(73)"},
	{myInt32(73), "myInt32(73)"},
	{myInt64(73), "myInt64(73)"},
	{myUint(73), "myUint(73)"},
	{myUint8(73), "myUint8(73)"},
	{myUint16(73), "myUint16(73)"},
	{myUint32(73), "myUint32(73)"},
	{myUint64(73), "myUint64(73)"},
	{myUintptr(73), "myUintptr(73)"},
	{myFloat32(12.639), "myFloat32(1.2639e+01)"},
	{myFloat64(12.639), "myFloat64(1.2639e+01)"},
	{myComplex64(6.2 + 2.7i), "myComplex64(6.2e+002.7e+00)"},
	{myComplex128(6.2 + 2.7i), "myComplex128(6.200e+002.700e+00)"},
	{myString("foo"), "myString(\"foo\")"},
}

func TestPanicToString(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			s := panicToString(tt.val)
			if !strings.HasPrefix(s, "runtime.") || s[len("runtime."):] != tt.str {
				t.Fatalf("expecting %s, got %s", tt.str, s)
			}
		})
	}
}
