// run

package main

import "testpkg"

func main() {
	a := testpkg.A
	testpkg.PrintInt(a)
}
