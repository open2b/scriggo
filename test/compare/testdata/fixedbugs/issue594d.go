// run

package main

func g() (string, string) {
	return "g1", "g2"
}

func f() (string, string) {
	return g()
	return "f1", "f2"
}

func main() {
	x, y := f()
	println(x, y)
}
