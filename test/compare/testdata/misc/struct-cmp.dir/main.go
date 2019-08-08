package main

import "fmt"
import "b"

var A1 = struct{ F int }{}

var A2 = struct{ f int }{}

func main() {
	fmt.Println(A1)
	fmt.Println(b.B1)
	fmt.Println(A1 == b.B1)

	// This comparison cannot be done (type checking error): types are
	// different. This is the expected behaviour.
	//
	// fmt.Println(A2 == b.B2)

	i1 := interface{}(A2)
	i2 := interface{}(b.B2)
	fmt.Println(i1 == i2)
}
