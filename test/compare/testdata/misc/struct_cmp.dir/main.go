package main

import "fmt"
import "struct_cmp.dir/b"

var A1 = struct{ F int }{}

var A2 = struct{ f int }{}

var A3 = struct {
	F int `k:"v"`
}{}

var A4 = struct {
	F int `k:"v2"`
}{}

func main() {
	fmt.Println(A1)
	fmt.Println(b.B1)
	fmt.Println(A1 == b.B1)

	// This comparison cannot be done (type checking error): types are
	// different. This is the expected behavior.
	//
	// fmt.Println(A2 == b.B2)

	i1 := interface{}(A2)
	i2 := interface{}(b.B2)
	fmt.Println(i1 == i2)

	fmt.Println(A3)
	fmt.Println(b.B3)
	fmt.Println(A3 == b.B3)

	// This comparison cannot be done (type checking error): types are
	// different. This is the expected behavior.
	//
	// fmt.Println(A4 == b.B3)

	i3 := interface{}(A1)
	i4 := interface{}(b.B3)
	fmt.Println(i3 == i4)

}
