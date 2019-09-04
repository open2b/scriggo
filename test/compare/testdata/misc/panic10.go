// paniccheck

// Test a runtime error in a predefined function.

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
	testpkg.RuntimeError()
}
