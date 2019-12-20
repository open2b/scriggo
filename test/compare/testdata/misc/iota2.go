// run

package main

import (
	"fmt"
)

const (
	A = 10
	B = iota
	C
	D, E = iota, iota
)

const (
	F    = iota
	G, H = I, F
	I    = iota + 3
)

func main() {
	fmt.Println(A, B, C, D, E, F, G, H, I)
}
