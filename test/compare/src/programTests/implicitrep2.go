// run

package main

import (
	"fmt"
)

const (
	A = iota
)

const (
	B = iota
	C
)

func main() {
	fmt.Print(A, B, C)
}
