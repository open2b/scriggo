// run

package main

import (
	"fmt"

	"github.com/open2b/scriggo/test/compare/testpkg"
)

type St struct{ F int }
type Sp *St

func main() {

	x := testpkg.St{}
	x.F = 1
	if x.F != 1 {
		panic(fmt.Sprintf("x.F: unexpected %d", x.F))
	}

	x2 := &x
	x2.F = 2
	if x2.F != 2 {
		panic(fmt.Sprintf("x2.F: unexpected %d", x2.F))
	}

	x3 := &testpkg.St{}
	x3.F = 3
	if x3.F != 3 {
		panic(fmt.Sprintf("x3.F: unexpected %d", x3.F))
	}

	testpkg.Sv.F = 1
	if testpkg.Sv.F != 1 {
		panic(fmt.Sprintf("testpkg.Sv.F: unexpected %d", testpkg.Sv.F))
	}

	x4 := struct{ F int }{}
	x4.F = 4
	if x4.F != 4 {
		panic(fmt.Sprintf("x4.F: unexpected %d", x4.F))
	}

	x5 := &struct{ F int }{}
	x5.F = 5
	if x5.F != 5 {
		panic(fmt.Sprintf("x5.F: unexpected %d", x5.F))
	}

	x6 := St{}
	x6.F = 6
	if x6.F != 6 {
		panic(fmt.Sprintf("x6.F: unexpected %d", x6.F))
	}

	x7 := &St{}
	x7.F = 7
	if x7.F != 7 {
		panic(fmt.Sprintf("x7.F: unexpected %d", x7.F))
	}

	x8 := Sp(&x6)
	x8.F = 8
	if x8.F != 8 {
		panic(fmt.Sprintf("x8.F: unexpected %d", x8.F))
	}
}
