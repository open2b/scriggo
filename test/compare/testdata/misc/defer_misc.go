// run

package main

import (
	"fmt"
	"testpkg"
)

var fa = testpkg.PrintInt

func fb(i int) { fmt.Print(i) }

var fc = func(i int) { fmt.Print(i) }

func main() {

	fd := testpkg.PrintInt

	fe := func(i int) { fmt.Print(i) }

	f := func() {

		defer fa(1)
		defer fb(2)
		defer fc(3)
		defer fd(4)
		defer fe(5)
		defer testpkg.PrintInt(6)
		defer func(i int) { fmt.Print(i) }(7)

	}

	defer func() {

		defer fa(1)
		defer fb(2)
		defer fc(3)
		defer fd(4)
		defer fe(5)
		defer testpkg.PrintInt(6)
		defer func(i int) { fmt.Print(i) }(7)

	}()

	defer f()

	defer fa(1)
	defer fb(2)
	defer fc(3)
	// See issue https://github.com/open2b/scriggo/issues/99.
	//defer fd(4)
	//defer fe(5)
	defer testpkg.PrintInt(6)
	defer func(i int) { fmt.Print(i) }(7)

}
