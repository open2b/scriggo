// run

package main

import (
	"fmt"
)

func main() {
	const (
		A = iota * 4
		B = iota * 2
		C = (iota + 3) * 2
	)
	fmt.Print(A, B, C)
}
