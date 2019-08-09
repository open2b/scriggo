// run

// TODO(Gianluca): add tests for predefined functions.

package main

import "fmt"

func f1(a ...int) {
	fmt.Println(a, len(a))
}

func f2(a string, b ...int) {
	fmt.Println(a, b, len(b))
}

func main() {
	f1([]int{}...)
	f1([]int{1, 2, 3}...)
	f2("a", []int{}...)
	f2("b", []int{10, 20, 30}...)
}
