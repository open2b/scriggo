//+build ignore

package main

var A = "global"

func init() {
	println(A)
	A = "init 1"
}

func F() {
	println(A)
	A = "F"
}

func init() {
	println(A)
	A = "init 2"
}

func main() {
	println(A)
	F()
	println(A)
}
