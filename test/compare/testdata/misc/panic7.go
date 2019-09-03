// skip

// paniccheck

// Test a program panic propagation.

package main

import (
	"testpkg"
)

func main() {
	testpkg.CallFunction(f)
}

func f() {
	panic("7")
}
