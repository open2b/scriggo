// run

package main

import "github.com/open2b/scriggo/test/compare/testpkg"

func main() {
	var a, b, e, c, d int

	a = 2 + 1
	b = 3 + 10
	e = 4
	c = testpkg.Sum(a, b)
	d = c

	_ = d
	_ = e
	return
}
