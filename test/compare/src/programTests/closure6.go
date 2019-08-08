// run

package main

import (
	"fmt"
)

func main() {
	A := interface{}(5)
	f := func() {
		fmt.Printf("%v (%T)", A, A)
	}
	f()
}
