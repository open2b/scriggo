// runcmp

package main

import "fmt"

var A = "global"

func init() {
	fmt.Println(A)
	A = "init 1"
}

func F() {
	fmt.Println(A)
	A = "F"
}

func init() {
	fmt.Println(A)
	A = "init 2"
}

func main() {
	fmt.Println(A)
	F()
	fmt.Println(A)
}
