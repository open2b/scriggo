// run

package main

import (
	"fmt"
)

func main() {
	const (
		A = 2 << iota
		B
		C
	)
	fmt.Print(A, B, C)
}
