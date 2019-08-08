// run

package main

import (
	"fmt"
	"testpkg"
)

func main() {
	a := 2
	a = testpkg.F11(9)
	fmt.Print(a)
	return
}
