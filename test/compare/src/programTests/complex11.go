// run

package main

import (
	"fmt"
	"testpkg"
)

func main() {
	c1 := testpkg.Complex128(1 + 2i)
	c2 := testpkg.Complex128(3281 - 12 - 32i + 1.43i)
	c3 := c1 + c2
	c4 := c1 - c2
	c5 := -(c1 * c2)
	fmt.Printf("%v %T, ", c3, c3)
	fmt.Printf("%v %T, ", c4, c4)
	fmt.Printf("%v %T", c5, c5)
}
