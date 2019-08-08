// run

package main

import "fmt"

func main() {
	var a, b, c, d int
	fmt.Print(a, b, c, d)
	a = 10
	b = a
	fmt.Print(a, b)
	a += 40
	fmt.Print(a)
	c = a
	fmt.Print(a, b, c)
	a -= 5
	fmt.Print(a)
	d = a
	fmt.Print(a, b, c, d)
}
