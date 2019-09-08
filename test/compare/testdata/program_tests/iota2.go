// run

package main

import (
	"fmt"
)

const (
	A = iota * 4
	B = iota * 2
	C = (iota + 3) * 2
)

func main() {
	fmt.Print(A, B, C)
}
