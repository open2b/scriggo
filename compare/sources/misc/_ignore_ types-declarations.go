//+build ignore

package main

import "fmt"

type GlobalType []int

var g GlobalType = GlobalType{1, 2, 3}

func main() {

	fmt.Println(g)

	{
		type T int
		var t T = T(0)
		_ = t
	}
	{
		type Int int
		type S []Int
		var s S = S{Int(0), Int(1)}
		fmt.Println(s)
	}
	{
		type S struct{}
		var s S = S{}
		fmt.Println(s)
	}
	{
		type S struct{ A, B int }
		var s S = S{10, 20}
		fmt.Println(s)
	}

	// TODO (Gianluca):
	// {
	// 	type F1 struct{ A int }
	// 	type F2 int
	// 	type S struct {
	// 		F1 F1
	// 		F2 F2
	// 	}
	// 	var s S = S{
	// 		F1: F1{A: 100},
	// 		F2: F2(int(10)),
	// 	}
	// 	fmt.Println(s)
	// }
}
