// run

package main

var i int

var V = func() {
	_ = i
}

func main() {
	V()
}
