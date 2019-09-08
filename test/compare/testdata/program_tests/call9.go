// run

package main

import (
	"fmt"
	"testpkg"
)

func main() {
	a := 2
	a = testpkg.G11(9)
	fmt.Print(a)
	return
}
