// run

package main

func f() (int, int) { return 10, 20 }

func g() (y int) {
	x, y := f()
	_ = x
	return
}

func main() {
	v := g()
	println(v)
}
