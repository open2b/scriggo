// run

package main

var V = func() int {
	return i + len(j)
}

var i int = 42

var j string = "jjj"

func main() {
	println(V())
}
