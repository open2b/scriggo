// skip

// paniccheck

// Test a fatal error propagation.

package main

import (
	"testpkg"
)

func main() {
	defer func() {
		panic("BUG2")
	}()
	testpkg.CallFunction(f)
	panic("BUG1")
}

func f() {
	testpkg.Fatal("9")
}
