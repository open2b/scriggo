// run

package main

import (
	"fmt"
)

var A = 1

func main() {
	fmt.Print(A)
	func() {
		fmt.Print(A)
		A := 20
		fmt.Print(A)
	}()
	fmt.Print(A)
}
