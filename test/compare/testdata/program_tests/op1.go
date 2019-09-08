// run

package main

func main() {
	T := true
	F := false

	var a, b, c, d bool

	a = !T
	b = !F
	c = !!F
	d = !!T

	_ = a
	_ = b
	_ = c
	_ = d
	return
}
