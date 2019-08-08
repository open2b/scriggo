// run

package main

import (
	"fmt"
	"testpkg"
)

func main() {
	c1 := testpkg.Complex128(1 + 2i)
	fmt.Print(c1)
	fmt.Printf("%T", c1)
}
