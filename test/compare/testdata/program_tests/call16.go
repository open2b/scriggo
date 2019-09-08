// run

package main

import "testpkg"

func main() {
	var a, b, c int

	a = testpkg.Inc(5)
	b = testpkg.Dec(a)
	c = testpkg.Inc(0) + testpkg.Dec(testpkg.Dec(testpkg.Dec(b)))

	_, _, _ = a, b, c
}
