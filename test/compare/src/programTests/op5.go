// run

package main

func main() {
	T := true
	F := false

	var a, b, c, d, e, f, g, h bool

	a = T && T
	b = T && F
	c = F && T
	d = F && F

	e = T || T
	f = T || F
	g = F || T
	h = F || F

	_ = a
	_ = b
	_ = c
	_ = d
	_ = e
	_ = f
	_ = g
	_ = h
}
