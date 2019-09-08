// run

package main

import (
	"fmt"
)

func main() {
	const A = iota
	const B = iota
	const (
		C = iota
		D = iota
	)
	fmt.Print(A, B, C, D)
}
