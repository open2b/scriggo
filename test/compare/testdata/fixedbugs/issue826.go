// run

package main

type x struct{}

var s2 struct {
	a, b, c int
	d       x
}

func main() {
	_ = &s2
}
