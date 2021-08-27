// run

package main

import (
	"fmt"

	"github.com/open2b/scriggo/test/compare/testpkg"
)

func main() {
	var i1, i2 int
	var s1, s2 string

	i1 = 3
	s1 = "hey"

	s1, i1 = testpkg.Swap(i1, s1)
	s2, i2 = testpkg.Swap(i1, s1)

	fmt.Print(s1, s2, i1, i2)
}
