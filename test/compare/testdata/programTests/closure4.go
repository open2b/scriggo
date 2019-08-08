// run

package main

import (
	"fmt"
)

func main() {
	A := 10
	f := func() {
		a := A
		fmt.Print("f: ", a)
	}
	f()
}
