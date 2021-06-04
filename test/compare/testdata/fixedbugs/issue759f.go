// run

package main

var V = struct{ Field func() string }{
	Field: F,
}

func F() string { return "F" }

func main() {
	println(V.Field())
}
