// run

package main

var i int = 42

var V = func() int {
	return i
}

func main() {
	println(V())
}
