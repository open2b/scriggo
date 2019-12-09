// skip : it is not a test but generates a test.

// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
)

// This program generates tests to verify that zeroing operations
// zero the data they are supposed to and clobber no adjacent values.

// run as `go run zeroGen.go`.  A file called zero.go
// will be written into the parent directory containing the tests.

var sizes = [...]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 15, 16, 17, 23, 24, 25, 31, 32, 33, 63, 64, 65, 1023, 1024, 1025}
var usizes = [...]int{8, 16, 24, 32, 64, 256}

func main() {
	w := new(bytes.Buffer)
	fmt.Fprintf(w, "// run\n\n")
	fmt.Fprintf(w, "// Code generated by gen/zeroGen.go. DO NOT EDIT.\n\n")
	fmt.Fprintf(w, "package main\n")
	fmt.Fprintf(w, "import \"log\"\n")

	for _, s := range sizes {
		// type for test
		fmt.Fprintf(w, "type Z%d struct {\n", s)
		fmt.Fprintf(w, "  pre [8]byte\n")
		fmt.Fprintf(w, "  mid [%d]byte\n", s)
		fmt.Fprintf(w, "  post [8]byte\n")
		fmt.Fprintf(w, "}\n")

		// function being tested
		fmt.Fprintf(w, "//go:noinline\n")
		fmt.Fprintf(w, "func zero%d_ssa(x *[%d]byte) {\n", s, s)
		fmt.Fprintf(w, "  *x = [%d]byte{}\n", s)
		fmt.Fprintf(w, "}\n")

		// testing harness
		fmt.Fprintf(w, "func testZero%d() {\n", s)
		fmt.Fprintf(w, "  a := Z%d{[8]byte{255,255,255,255,255,255,255,255},[%d]byte{", s, s)
		for i := 0; i < s; i++ {
			fmt.Fprintf(w, "255,")
		}
		fmt.Fprintf(w, "},[8]byte{255,255,255,255,255,255,255,255}}\n")
		fmt.Fprintf(w, "  zero%d_ssa(&a.mid)\n", s)
		fmt.Fprintf(w, "  want := Z%d{[8]byte{255,255,255,255,255,255,255,255},[%d]byte{", s, s)
		for i := 0; i < s; i++ {
			fmt.Fprintf(w, "0,")
		}
		fmt.Fprintf(w, "},[8]byte{255,255,255,255,255,255,255,255}}\n")
		fmt.Fprintf(w, "  if a != want {\n")
		fmt.Fprintf(w, "    log.Printf(\"zero%d got=%%v, want %%v\\n\", a, want)\n", s)
		fmt.Fprintf(w, "  }\n")
		fmt.Fprintf(w, "}\n")
	}

	for _, s := range usizes {
		// type for test
		fmt.Fprintf(w, "type Z%du1 struct {\n", s)
		fmt.Fprintf(w, "  b   bool\n")
		fmt.Fprintf(w, "  val [%d]byte\n", s)
		fmt.Fprintf(w, "}\n")

		fmt.Fprintf(w, "type Z%du2 struct {\n", s)
		fmt.Fprintf(w, "  i   uint16\n")
		fmt.Fprintf(w, "  val [%d]byte\n", s)
		fmt.Fprintf(w, "}\n")

		// function being tested
		fmt.Fprintf(w, "//go:noinline\n")
		fmt.Fprintf(w, "func zero%du1_ssa(t *Z%du1) {\n", s, s)
		fmt.Fprintf(w, "  t.val = [%d]byte{}\n", s)
		fmt.Fprintf(w, "}\n")

		// function being tested
		fmt.Fprintf(w, "//go:noinline\n")
		fmt.Fprintf(w, "func zero%du2_ssa(t *Z%du2) {\n", s, s)
		fmt.Fprintf(w, "  t.val = [%d]byte{}\n", s)
		fmt.Fprintf(w, "}\n")

		// testing harness
		fmt.Fprintf(w, "func testZero%du() {\n", s)
		fmt.Fprintf(w, "  a := Z%du1{false, [%d]byte{", s, s)
		for i := 0; i < s; i++ {
			fmt.Fprintf(w, "255,")
		}
		fmt.Fprintf(w, "}}\n")
		fmt.Fprintf(w, "  zero%du1_ssa(&a)\n", s)
		fmt.Fprintf(w, "  want := Z%du1{false, [%d]byte{", s, s)
		for i := 0; i < s; i++ {
			fmt.Fprintf(w, "0,")
		}
		fmt.Fprintf(w, "}}\n")
		fmt.Fprintf(w, "  if a != want {\n")
		fmt.Fprintf(w, "    log.Printf(\"zero%du2 got=%%v, want %%v\\n\", a, want)\n", s)
		fmt.Fprintf(w, "  }\n")
		fmt.Fprintf(w, "  b := Z%du2{15, [%d]byte{", s, s)
		for i := 0; i < s; i++ {
			fmt.Fprintf(w, "255,")
		}
		fmt.Fprintf(w, "}}\n")
		fmt.Fprintf(w, "  zero%du2_ssa(&b)\n", s)
		fmt.Fprintf(w, "  wantb := Z%du2{15, [%d]byte{", s, s)
		for i := 0; i < s; i++ {
			fmt.Fprintf(w, "0,")
		}
		fmt.Fprintf(w, "}}\n")
		fmt.Fprintf(w, "  if b != wantb {\n")
		fmt.Fprintf(w, "    log.Printf(\"zero%du2 got=%%v, want %%v\\n\", b, wantb)\n", s)
		fmt.Fprintf(w, "  }\n")
		fmt.Fprintf(w, "}\n")
	}

	// boilerplate at end
	fmt.Fprintf(w, "\nfunc main() {\n")
	for _, s := range sizes {
		fmt.Fprintf(w, "  testZero%d()\n", s)
	}
	for _, s := range usizes {
		fmt.Fprintf(w, "  testZero%du()\n", s)
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
	err = ioutil.WriteFile("../zero.go", src, 0666)
	if err != nil {
		log.Fatalf("can't write output: %v\n", err)
	}
}
