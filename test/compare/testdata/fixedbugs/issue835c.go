// run

package main

func main() {
	g := func() (y int) {
		x, y := 10, 20
		_ = x
		return
	}
	v := g()
	println(v)
}
