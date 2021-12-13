// run

package main

import "fmt"

var x complex64

func main() {
	x = 3i
	y := x
	x = 0
	fmt.Printf("x: %#v\n", x)
	fmt.Printf("y: %#v\n", y)
	x = 32
	fmt.Printf("x: %#v\n", x)
	fmt.Printf("y: %#v\n", y)
}
