// skip : it is not a test but generates a test.

// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This program generates a test to verify that the standard comparison
// operators properly handle one const operand. The test file should be
// generated with a known working version of go.
// launch with `go run cmpConstGen.go` a file called cmpConst.go
// will be written into the parent directory containing the tests

package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"math/big"
	"sort"
)

const (
	maxU64 = (1 << 64) - 1
	maxU32 = (1 << 32) - 1
	maxU16 = (1 << 16) - 1
	maxU8  = (1 << 8) - 1

	maxI64 = (1 << 63) - 1
	maxI32 = (1 << 31) - 1
	maxI16 = (1 << 15) - 1
	maxI8  = (1 << 7) - 1

	minI64 = -(1 << 63)
	minI32 = -(1 << 31)
	minI16 = -(1 << 15)
	minI8  = -(1 << 7)
)

func cmp(left *big.Int, op string, right *big.Int) bool {
	switch left.Cmp(right) {
	case -1: // less than
		return op == "<" || op == "<=" || op == "!="
	case 0: // equal
		return op == "==" || op == "<=" || op == ">="
	case 1: // greater than
		return op == ">" || op == ">=" || op == "!="
	}
	panic("unexpected comparison value")
}

func inRange(typ string, val *big.Int) bool {
	min, max := &big.Int{}, &big.Int{}
	switch typ {
	case "uint64":
		max = max.SetUint64(maxU64)
	case "uint32":
		max = max.SetUint64(maxU32)
	case "uint16":
		max = max.SetUint64(maxU16)
	case "uint8":
		max = max.SetUint64(maxU8)
	case "int64":
		min = min.SetInt64(minI64)
		max = max.SetInt64(maxI64)
	case "int32":
		min = min.SetInt64(minI32)
		max = max.SetInt64(maxI32)
	case "int16":
		min = min.SetInt64(minI16)
		max = max.SetInt64(maxI16)
	case "int8":
		min = min.SetInt64(minI8)
		max = max.SetInt64(maxI8)
	default:
		panic("unexpected type")
	}
	return cmp(min, "<=", val) && cmp(val, "<=", max)
}

func getValues(typ string) []*big.Int {
	Uint := func(v uint64) *big.Int { return big.NewInt(0).SetUint64(v) }
	Int := func(v int64) *big.Int { return big.NewInt(0).SetInt64(v) }
	values := []*big.Int{
		// limits
		Uint(maxU64),
		Uint(maxU64 - 1),
		Uint(maxI64 + 1),
		Uint(maxI64),
		Uint(maxI64 - 1),
		Uint(maxU32 + 1),
		Uint(maxU32),
		Uint(maxU32 - 1),
		Uint(maxI32 + 1),
		Uint(maxI32),
		Uint(maxI32 - 1),
		Uint(maxU16 + 1),
		Uint(maxU16),
		Uint(maxU16 - 1),
		Uint(maxI16 + 1),
		Uint(maxI16),
		Uint(maxI16 - 1),
		Uint(maxU8 + 1),
		Uint(maxU8),
		Uint(maxU8 - 1),
		Uint(maxI8 + 1),
		Uint(maxI8),
		Uint(maxI8 - 1),
		Uint(0),
		Int(minI8 + 1),
		Int(minI8),
		Int(minI8 - 1),
		Int(minI16 + 1),
		Int(minI16),
		Int(minI16 - 1),
		Int(minI32 + 1),
		Int(minI32),
		Int(minI32 - 1),
		Int(minI64 + 1),
		Int(minI64),

		// other possibly interesting values
		Uint(1),
		Int(-1),
		Uint(0xff << 56),
		Uint(0xff << 32),
		Uint(0xff << 24),
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Cmp(values[j]) == -1 })
	var ret []*big.Int
	for _, val := range values {
		if !inRange(typ, val) {
			continue
		}
		ret = append(ret, val)
	}
	return ret
}

func sigString(v *big.Int) string {
	var t big.Int
	t.Abs(v)
	if v.Sign() == -1 {
		return "neg" + t.String()
	}
	return t.String()
}

func main() {
	types := []string{
		"uint64", "uint32", "uint16", "uint8",
		"int64", "int32", "int16", "int8",
	}

	w := new(bytes.Buffer)
	fmt.Fprintf(w, "// run\n\n")
	fmt.Fprintf(w, "// Code generated by gen/cmpConstGen.go. DO NOT EDIT.\n\n")
	fmt.Fprintf(w, "package main;\n")
	fmt.Fprintf(w, "import (\"log\"; \"reflect\"; \"runtime\";)\n")
	fmt.Fprintf(w, "// results show the expected result for the elements left of, equal to and right of the index.\n")
	fmt.Fprintf(w, "type result struct{l, e, r bool}\n")
	fmt.Fprintf(w, "var (\n")
	fmt.Fprintf(w, "	eq = result{l: false, e: true, r: false}\n")
	fmt.Fprintf(w, "	ne = result{l: true, e: false, r: true}\n")
	fmt.Fprintf(w, "	lt = result{l: true, e: false, r: false}\n")
	fmt.Fprintf(w, "	le = result{l: true, e: true, r: false}\n")
	fmt.Fprintf(w, "	gt = result{l: false, e: false, r: true}\n")
	fmt.Fprintf(w, "	ge = result{l: false, e: true, r: true}\n")
	fmt.Fprintf(w, ")\n")

	operators := []struct{ op, name string }{
		{"<", "lt"},
		{"<=", "le"},
		{">", "gt"},
		{">=", "ge"},
		{"==", "eq"},
		{"!=", "ne"},
	}

	for _, typ := range types {
		// generate a slice containing valid values for this type
		fmt.Fprintf(w, "\n// %v tests\n", typ)
		values := getValues(typ)
		fmt.Fprintf(w, "var %v_vals = []%v{\n", typ, typ)
		for _, val := range values {
			fmt.Fprintf(w, "%v,\n", val.String())
		}
		fmt.Fprintf(w, "}\n")

		// generate test functions
		for _, r := range values {
			// TODO: could also test constant on lhs.
			sig := sigString(r)
			for _, op := range operators {
				// no need for go:noinline because the function is called indirectly
				fmt.Fprintf(w, "func %v_%v_%v(x %v) bool { return x %v %v; }\n", op.name, sig, typ, typ, op.op, r.String())
			}
		}

		// generate a table of test cases
		fmt.Fprintf(w, "var %v_tests = []struct{\n", typ)
		fmt.Fprintf(w, "	idx int // index of the constant used\n")
		fmt.Fprintf(w, "	exp result // expected results\n")
		fmt.Fprintf(w, "	fn  func(%v) bool\n", typ)
		fmt.Fprintf(w, "}{\n")
		for i, r := range values {
			sig := sigString(r)
			for _, op := range operators {
				fmt.Fprintf(w, "{idx: %v,", i)
				fmt.Fprintf(w, "exp: %v,", op.name)
				fmt.Fprintf(w, "fn:  %v_%v_%v},\n", op.name, sig, typ)
			}
		}
		fmt.Fprintf(w, "}\n")
	}

	// emit the main function, looping over all test cases
	fmt.Fprintf(w, "// Test results for comparison operations against constants.\n")
	fmt.Fprintf(w, "func main() {\n")
	for _, typ := range types {
		fmt.Fprintf(w, "for i, test := range %v_tests {\n", typ)
		fmt.Fprintf(w, "	for j, x := range %v_vals {\n", typ)
		fmt.Fprintf(w, "		want := test.exp.l\n")
		fmt.Fprintf(w, "		if j == test.idx {\nwant = test.exp.e\n}")
		fmt.Fprintf(w, "		else if j > test.idx {\nwant = test.exp.r\n}\n")
		fmt.Fprintf(w, "		if test.fn(x) != want {\n")
		fmt.Fprintf(w, "			fn := runtime.FuncForPC(reflect.ValueOf(test.fn).Pointer()).Name()\n")
		fmt.Fprintf(w, "			log.Fatalf(\"test failed: %%v(%%v) != %%v [type=%v i=%%v j=%%v idx=%%v]\", fn, x, want, i, j, test.idx)\n", typ)
		fmt.Fprintf(w, "		}\n")
		fmt.Fprintf(w, "	}\n")
		fmt.Fprintf(w, "}\n")
	}
	fmt.Fprintf(w, "}\n")

	// gofmt result
	b := w.Bytes()
	src, err := format.Source(b)
	if err != nil {
		fmt.Printf("%s\n", b)
		panic(err)
	}

	// write to file
	err = ioutil.WriteFile("../cmpConst.go", src, 0666)
	if err != nil {
		log.Fatalf("can't write output: %v\n", err)
	}
}
