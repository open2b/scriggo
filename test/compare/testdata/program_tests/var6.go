// run

package main

import "github.com/open2b/scriggo/test/compare/testpkg"

func main() {
	testpkg.PrintInt(testpkg.B)
	testpkg.B = 7
	testpkg.PrintString("->")
	testpkg.PrintInt(testpkg.B)
}
