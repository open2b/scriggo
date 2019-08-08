// run

package main

import (
	"fmt"
)

var A = 1

func main() {
	fmt.Print(A)
	A = 2
	func() {
		fmt.Print(A)
		A = 3
		fmt.Print(A)
		f := func() {
			A = 4
			fmt.Print(A)
		}
		fmt.Print(A)
		f()
	}()
	fmt.Print(A)
}
