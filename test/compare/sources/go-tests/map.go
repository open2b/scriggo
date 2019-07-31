// run

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Test maps, almost exhaustively.
// Complexity (linearity) test is in maplinear.go.

package main

import (
	"fmt"
	"math"
)

func testbasic() {
	// // Test a map literal.
	// mlit := map[string]int{"0": 0, "1": 1, "2": 2, "3": 3, "4": 4}
	// for i := 0; i < len(mlit); i++ {
	// 	s := string([]byte{byte(i) + '0'})
	// 	if mlit[s] != i {
	// 		panic(fmt.Sprintf("mlit[%s] = %d\n", s, mlit[s]))
	// 	}
	// }
}

func testfloat() {
	// Test floating point numbers in maps.
	// Two map keys refer to the same entry if the keys are ==.
	// The special cases, then, are that +0 == -0 and that NaN != NaN.

	{
		var (
			pz   = float32(0)
			nz   = math.Float32frombits(1 << 31)
			nana = float32(math.NaN())
			nanb = math.Float32frombits(math.Float32bits(nana) ^ 2)
		)

		m := map[float32]string{
			pz:   "+0",
			nana: "NaN",
			nanb: "NaN",
		}
		if m[pz] != "+0" {
			panic(fmt.Sprintln("float32 map cannot read back m[+0]:", m[pz]))
		}
		if m[nz] != "+0" {
			fmt.Sprintln("float32 map does not treat", pz, "and", nz, "as equal for read")
			panic(fmt.Sprintln("float32 map does not treat -0 and +0 as equal for read"))
		}
		m[nz] = "-0"
		if m[pz] != "-0" {
			panic(fmt.Sprintln("float32 map does not treat -0 and +0 as equal for write"))
		}
		if _, ok := m[nana]; ok {
			panic(fmt.Sprintln("float32 map allows NaN lookup (a)"))
		}
		if _, ok := m[nanb]; ok {
			panic(fmt.Sprintln("float32 map allows NaN lookup (b)"))
		}
		if len(m) != 3 {
			panic(fmt.Sprintln("float32 map should have 3 entries:", m))
		}
		m[nana] = "NaN"
		m[nanb] = "NaN"
		if len(m) != 5 {
			panic(fmt.Sprintln("float32 map should have 5 entries:", m))
		}
	}

	{
		var (
			pz   = float64(0)
			nz   = math.Float64frombits(1 << 63)
			nana = float64(math.NaN())
			nanb = math.Float64frombits(math.Float64bits(nana) ^ 2)
		)

		m := map[float64]string{
			pz:   "+0",
			nana: "NaN",
			nanb: "NaN",
		}
		if m[nz] != "+0" {
			panic(fmt.Sprintln("float64 map does not treat -0 and +0 as equal for read"))
		}
		m[nz] = "-0"
		if m[pz] != "-0" {
			panic(fmt.Sprintln("float64 map does not treat -0 and +0 as equal for write"))
		}
		if _, ok := m[nana]; ok {
			panic(fmt.Sprintln("float64 map allows NaN lookup (a)"))
		}
		if _, ok := m[nanb]; ok {
			panic(fmt.Sprintln("float64 map allows NaN lookup (b)"))
		}
		if len(m) != 3 {
			panic(fmt.Sprintln("float64 map should have 3 entries:", m))
		}
		m[nana] = "NaN"
		m[nanb] = "NaN"
		if len(m) != 5 {
			panic(fmt.Sprintln("float64 map should have 5 entries:", m))
		}
	}

	// TODO(Gianluca): https://github.com/open2b/scriggo/issues/172
	// {
	// 	var (
	// 		pz   = complex64(0)
	// 		nz   = complex(0, math.Float32frombits(1<<31))
	// 		nana = complex(5, float32(math.NaN()))
	// 		nanb = complex(5, math.Float32frombits(math.Float32bits(float32(math.NaN()))^2))
	// 	)

	// 	m := map[complex64]string{
	// 		pz:   "+0",
	// 		nana: "NaN",
	// 		nanb: "NaN",
	// 	}
	// 	if m[nz] != "+0" {
	// 		panic(fmt.Sprintln("complex64 map does not treat -0 and +0 as equal for read"))
	// 	}
	// 	m[nz] = "-0"
	// 	if m[pz] != "-0" {
	// 		panic(fmt.Sprintln("complex64 map does not treat -0 and +0 as equal for write"))
	// 	}
	// 	if _, ok := m[nana]; ok {
	// 		panic(fmt.Sprintln("complex64 map allows NaN lookup (a)"))
	// 	}
	// 	if _, ok := m[nanb]; ok {
	// 		panic(fmt.Sprintln("complex64 map allows NaN lookup (b)"))
	// 	}
	// 	if len(m) != 3 {
	// 		panic(fmt.Sprintln("complex64 map should have 3 entries:", m))
	// 	}
	// 	m[nana] = "NaN"
	// 	m[nanb] = "NaN"
	// 	if len(m) != 5 {
	// 		panic(fmt.Sprintln("complex64 map should have 5 entries:", m))
	// 	}
	// }

	// {
	// 	var (
	// 		pz   = complex128(0)
	// 		nz   = complex(0, math.Float64frombits(1<<63))
	// 		nana = complex(5, float64(math.NaN()))
	// 		nanb = complex(5, math.Float64frombits(math.Float64bits(float64(math.NaN()))^2))
	// 	)

	// 	m := map[complex128]string{
	// 		pz:   "+0",
	// 		nana: "NaN",
	// 		nanb: "NaN",
	// 	}
	// 	if m[nz] != "+0" {
	// 		panic(fmt.Sprintln("complex128 map does not treat -0 and +0 as equal for read"))
	// 	}
	// 	m[nz] = "-0"
	// 	if m[pz] != "-0" {
	// 		panic(fmt.Sprintln("complex128 map does not treat -0 and +0 as equal for write"))
	// 	}
	// 	if _, ok := m[nana]; ok {
	// 		panic(fmt.Sprintln("complex128 map allows NaN lookup (a)"))
	// 	}
	// 	if _, ok := m[nanb]; ok {
	// 		panic(fmt.Sprintln("complex128 map allows NaN lookup (b)"))
	// 	}
	// 	if len(m) != 3 {
	// 		panic(fmt.Sprintln("complex128 map should have 3 entries:", m))
	// 	}
	// 	m[nana] = "NaN"
	// 	m[nanb] = "NaN"
	// 	if len(m) != 5 {
	// 		panic(fmt.Sprintln("complex128 map should have 5 entries:", m))
	// 	}
	// }
}

func testnan() {
	n := 500
	m := map[float64]int{}
	nan := math.NaN()
	for i := 0; i < n; i++ {
		m[nan] = 1
	}
	if len(m) != n {
		panic("wrong size map after nan insertion")
	}
	iters := 0
	for k, v := range m {
		iters++
		if !math.IsNaN(k) {
			panic("not NaN")
		}
		if v != 1 {
			panic("wrong value")
		}
	}
	if iters != n {
		panic("wrong number of nan range iters")
	}
}

func main() {
	testbasic()
	testfloat()
	testnan()
}
