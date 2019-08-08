// run

package main

import (
	"fmt"
)

func main() {
	A := 10
	B := "hey"
	f := func() {
		a := A
		fmt.Print("f: ", a, ",")
		b := B
		fmt.Print("f: ", b)
	}
	f()
}
