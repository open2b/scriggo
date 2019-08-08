// run

package main

import (
	"fmt"
)

func main() {
	A := 10
	f := func() {
		fmt.Print("f: ", A, ", ")
		g := func() {
			fmt.Print("g: ", A)
		}
		g()
	}
	f()
}
