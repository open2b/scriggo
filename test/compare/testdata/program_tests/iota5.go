// run

package main

import (
	"fmt"
)

func main() {
	const (
		A = (iota * iota) + 10
		B = A * (iota * 2.341)
		C = A + B + iota/2.312
	)
	fmt.Print(A, B, C)
}
