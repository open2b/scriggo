// run

package main

import (
	"fmt"
)

func main() {
	const (
		A = 10 + iota + (iota * 2)
		B
		C
	)
	fmt.Print(A, B, C)
}
