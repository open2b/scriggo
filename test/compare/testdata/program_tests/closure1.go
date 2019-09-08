// run

package main

import (
	"fmt"
)

func main() {
	A := 10
	B := 20
	f1 := func() {
		fmt.Print(A)
		A = 10
	}
	f2 := func() {
		fmt.Print(B)
		A = B + 2
	}
	f3 := func() {
		fmt.Print(A + B)
		B = A + B
	}
	f1()
	f2()
	f3()
}
