// run

// This test has been extrapolated form test "src/cmd/compile/internal/gc/float_test.go"
// in the Go repository and has been adapted not to use the "testing" package.
// The original file has copyright:

// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"math"
)

// Signaling NaN values as constants.
const (
	snan32bits uint32 = 0x7f800001
	snan64bits uint64 = 0x7ff0000000000001
)

// Signaling NaNs as variables.
var snan32bitsVar uint32 = snan32bits
var snan64bitsVar uint64 = snan64bits

func main() {
	TestFloatSignalingNaN()
	TestFloatSignalingNaNConversion()
	TestFloatSignalingNaNConversionConst()
}

func TestFloatSignalingNaN() {
	// Make sure we generate a signaling NaN from a constant properly.
	// See issue 36400.
	f32 := math.Float32frombits(snan32bits)
	g32 := math.Float32frombits(snan32bitsVar)
	x32 := math.Float32bits(f32)
	y32 := math.Float32bits(g32)
	if x32 != y32 {
		panic(fmt.Errorf("got %x, want %x (diff=%x)", x32, y32, x32^y32))
	}

	f64 := math.Float64frombits(snan64bits)
	g64 := math.Float64frombits(snan64bitsVar)
	x64 := math.Float64bits(f64)
	y64 := math.Float64bits(g64)
	if x64 != y64 {
		panic(fmt.Errorf("got %x, want %x (diff=%x)", x64, y64, x64^y64))
	}
}

func TestFloatSignalingNaNConversion() {
	// Test to make sure when we convert a signaling NaN, we get a NaN.
	// (Ideally we want a quiet NaN, but some platforms don't agree.)
	// See issue 36399.
	s32 := math.Float32frombits(snan32bitsVar)
	if s32 == s32 {
		panic(fmt.Errorf("converting a NaN did not result in a NaN"))
	}
	s64 := math.Float64frombits(snan64bitsVar)
	if s64 == s64 {
		panic(fmt.Errorf("converting a NaN did not result in a NaN"))
	}
}

func TestFloatSignalingNaNConversionConst() {
	// Test to make sure when we convert a signaling NaN, it converts to a NaN.
	// (Ideally we want a quiet NaN, but some platforms don't agree.)
	// See issue 36399 and 36400.
	s32 := math.Float32frombits(snan32bits)
	if s32 == s32 {
		panic(fmt.Errorf("converting a NaN did not result in a NaN"))
	}
	s64 := math.Float64frombits(snan64bits)
	if s64 == s64 {
		panic(fmt.Errorf("converting a NaN did not result in a NaN"))
	}
}
