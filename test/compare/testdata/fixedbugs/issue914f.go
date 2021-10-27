// run

package main

import "fmt"

var x *int

func main() {
	x = new(int)
	y := x
	fmt.Println(x == nil, y == nil)
	x = nil
	fmt.Println(x == nil, y == nil)
}
