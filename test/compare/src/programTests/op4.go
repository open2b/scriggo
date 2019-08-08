// run

package main

import "fmt"

func main() {
	var a, b int
	var l, g, le, ge, e, ne bool

	a = 1
	b = 2

	l = a < b
	g = a > b
	le = a <= b
	ge = a >= b
	e = a == b
	ne = a != b

	fmt.Print(l)
	fmt.Print(g)
	fmt.Print(le)
	fmt.Print(ge)
	fmt.Print(e)
	fmt.Print(ne)
}
