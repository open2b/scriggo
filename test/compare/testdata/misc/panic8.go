// paniccheck

// Test a runtime error propagation.

package main

import (
	"runtime"

	"github.com/open2b/scriggo/test/compare/testpkg"
)

func main() {
	defer func() {
		r := recover()
		if _, ok := r.(runtime.Error); !ok {
			panic("BUG")
		}
		panic(r)
	}()
	testpkg.CallFunction(f)
}

func f() {
	var a = 0
	_ = 1 / a
}
