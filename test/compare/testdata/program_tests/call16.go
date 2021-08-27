// run

package main

import "github.com/open2b/scriggo/test/compare/testpkg"

func main() {
	var a, b, c int

	a = testpkg.Inc(5)
	b = testpkg.Dec(a)
	c = testpkg.Inc(0) + testpkg.Dec(testpkg.Dec(testpkg.Dec(b)))

	_, _, _ = a, b, c
}
