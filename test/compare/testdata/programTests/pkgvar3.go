// run

package main

import "fmt"

var A = 3

func SetA() {
	A = 4
}

func main() {
	a := A
	fmt.Print(a)
	SetA()
	a = A
	fmt.Print(a)
}
