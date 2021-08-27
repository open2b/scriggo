// paniccheck

// Test a program panic propagation.

package main

import (
	"github.com/open2b/scriggo/test/compare/testpkg"
)

func main() {
	testpkg.CallFunction(f)
}

func f() {
	panic("7")
}
