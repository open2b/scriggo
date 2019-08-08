// run

package main

import "testpkg"

func main() {
	testpkg.PrintInt(testpkg.B)
	testpkg.B = 7
	testpkg.PrintString("->")
	testpkg.PrintInt(testpkg.B)
}
