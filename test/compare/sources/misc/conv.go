// runcmp

// Copyright (c) 2009 The Go Authors. All rights reserved.

// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:

//    * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//    * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//    * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.

// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

// Some of the tests below are a readaptation of
// https://golang.org/ref/spec#Conversions.

package main

import "math"

func assert(cond bool, msg string) {
	if !cond {
		panic("assertion failed: " + msg)
	}
}

func compareSliceByte(s1, s2 []byte) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := 0; i < len(s1); i++ {
		if s1[i] != s2[i] {
			return false
		}
	}
	return true
}

func compareSliceRune(s1, s2 []rune) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := 0; i < len(s1); i++ {
		if s1[i] != s2[i] {
			return false
		}
	}
	return true
}

func main() {

	// Conversions to and from a string type.

	// 1. Converting a signed or unsigned integer value to a string type.
	s1 := string('a')
	assert(string(s1) == "a", "rune -> string")
	s2 := -1
	assert(string(s2) == "\ufffd", "negative codepoint -> string")
	s3 := 0xf8
	assert(string(s3) == "ø", "hex codepoint -> string")

	// 2. Converting a slice of bytes to a string type.
	s4 := []byte{'h', 'e', 'l', 'l', '\xc3', '\xb8'}
	assert(string(s4) == "hellø", "[]byte -> string")
	s5 := []byte{}
	assert(string(s5) == "", "empty []byte -> string")
	s6 := []byte(nil)
	assert(string(s6) == "", "nil []byte -> string")

	// 3. Converting a slice of runes to a string type.
	s7 := []rune{0x767d, 0x9d6c, 0x7fd4}
	assert(string(s7) == "白鵬翔", "[]rune -> string")
	s8 := []byte{}
	assert(string(s8) == "", "empty []rune -> string")
	s9 := []byte(nil)
	assert(string(s9) == "", "nil []rune -> string")

	// 4. Converting a value of a string type to a slice of bytes type.
	s10 := string("hellø")
	assert(compareSliceByte([]byte(s10), []byte{'h', 'e', 'l', 'l', '\xc3', '\xb8'}), "string -> []byte")
	s11 := string("")
	assert(compareSliceByte([]byte(s11), []byte{}), "empty string -> []byte")

	// 5. Converting a value of a string type to a slice of runes type.
	s12 := string("白鵬翔")
	assert(compareSliceRune([]rune(s12), []rune{0x767d, 0x9d6c, 0x7fd4}), "string -> []rune")
	s13 := string("")
	assert(compareSliceRune([]rune(s13), []rune{}), "empty string -> []rune")

	// Conversions between numeric types.

	// 1. Converting between integer types.
	i1 := int64(1234)
	assert(int64(i1) == 1234, "int64 -> int64")
	i2 := int64(math.MaxInt64)
	assert(int64(i2) == math.MaxInt64, "int64 -> int64")
	i3 := int64(math.MaxInt32 + 1)
	assert(int32(i3) == math.MinInt32, "int64 -> int32 (overflow)")
	i4 := int32(math.MaxInt32)
	assert(int64(i4) == math.MaxInt32, "int32 -> int64")
	i5 := int32(1234)
	assert(uint32(i5) == 1234, "int32 -> uint32")
	i6 := int32(math.MaxInt32)
	assert(uint32(i6) == math.MaxInt32, "int32 -> uint32")
	i7 := uint64(math.MaxUint64)
	assert(int64(i7) == -1, "uint64 -> int64 (overflow)")
	i8 := int16(200)
	assert(uint8(i8) == 200, "int16 -> uint8")

	// 2. Converting a floating-point number to an integer.
	f1 := float64(12)
	assert(int64(f1) == 12, "float64 -> int64")
	f2 := float64(12.00000)
	assert(int64(f2) == 12, "float64 -> int64")
	f3 := float64(-20)
	assert(int64(f3) == -20, "float64 -> int64")
	f4 := float64(20.5)
	assert(int64(f4) == 20, "float64 -> int64 (truncation towards zero)")
	f5 := float64(-12.7)
	assert(int64(f5) == -12, "float64 -> int64 (truncation towards zero)")
	f6 := float64(12)
	assert(uint64(f6) == 12, "float64 -> uint64")
	f7 := float64(12.00000)
	assert(uint64(f7) == 12, "float64 -> uint64")
	f8 := float64(-20)
	assert(uint64(f8) == 18446744073709551596, "float64 -> uint64")
	f9 := float64(20.5)
	assert(uint64(f9) == 20, "float64 -> uint64 (truncation towards zero)")
	f10 := float64(-12.7)
	assert(uint64(f10) == 18446744073709551604, "float64 -> uint64 (truncation towards zero)")

	// 3.a Converting an integer to a floating-point number.
	if1 := int64(20)
	assert(float64(if1) == 20, "int64 -> float64")
	if2 := int64(0)
	assert(float64(if2) == 0, "int64 -> float64 (zero)")
	if3 := int64(-100)
	assert(float64(if3) == -100, "int64 -> float64 (negative)")
	if4 := int8(4)
	assert(float64(if4) == 4, "int8 -> float64")
	if5 := int8(40)
	assert(float32(if5) == 40, "int8 -> float32")

	// 3.b. Converting a complex type to another complex type.
	// TODO(Gianluca).
}
