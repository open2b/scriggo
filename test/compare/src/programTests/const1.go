// run

package main

import "fmt"

func main() {
	var a, b int
	var c, d bool
	fmt.Print(a, b, c, d)
	a = 1
	b = 2
	fmt.Print(a, b, c, d)
	c = false
	d = 4 < 5
	fmt.Print(a, b, c, d)
}
