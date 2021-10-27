// run

package main

import "fmt"

var x *int

func main() {
	x = new(int)
	y := x
	x = nil
	fmt.Print(x == nil)
	fmt.Print(y == nil)
}
