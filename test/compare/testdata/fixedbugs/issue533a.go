// run

package main

func F() (int, int) {
	return 1, 2
}

func G() (int, int) {
	return F()
}

func main() {
	a, b := G()
	println(a, b)
}
