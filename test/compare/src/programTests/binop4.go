// run

package main

import (
	"fmt"
)

func doubleComplex(c complex128) complex128 {
	return 2 * c
}

func main() {
	fmt.Print(3 + 8i + doubleComplex(-2+0.2i))
	fmt.Print(doubleComplex(3+8i) + doubleComplex(-2+0.2i))
}
