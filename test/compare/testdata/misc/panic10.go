// paniccheck

// Test a runtime error in a native function.

package main

import (
	"github.com/open2b/scriggo/test/compare/testpkg"
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
