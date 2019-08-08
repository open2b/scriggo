// run

package main

import "fmt"

var A = F() + 3 + (2 * F())

func F() int {
	return 42
}

func main() {
	a := A
	fmt.Print(a)
}
