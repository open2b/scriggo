// run

package main

func g() (y int) {
	x, y := 10, 20
	_ = x
	return
}

func main() {
	v := g()
	println(v)
}
