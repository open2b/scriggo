// compile

package main

import "fmt"

type Int int

func Sum(a, b Int) Int {
	return a + b
}

type SumFunc func(a, b Int) Int

func main() {
	var a, b Int = 30, 12
	fmt.Println(Sum(a, b))

	// var s SumFunc = Sum
	// fmt.Println(s(20, 30))
	var _ func(A Int) = func(A Int) {}
	type String string
	var _ func(A Int) String = func(A Int) String { return "" }
}
