// run

package main

var V = struct{ F func() int }{
	func() int {
		return i * 3
	},
}

var i int = 42

func main() {
	println(V.F())
}
