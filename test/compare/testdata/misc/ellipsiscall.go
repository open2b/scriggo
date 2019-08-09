// run

package main

import "fmt"

func f1(a ...int) {
	fmt.Println(a, len(a))
}

func f2(a string, b ...int) {
	fmt.Println(a, b, len(b))
}

func main() {
	// Functions defined in Scriggo.
	f1([]int{}...)
	f1([]int{1, 2, 3}...)
	f2("a", []int{}...)
	f2("b", []int{10, 20, 30}...)

	// Predefined functions.
	s1 := []interface{}{1, 2, 3}
	fmt.Println(s1...)
	s2 := []interface{}{}
	fmt.Println(s2...)
	s3 := []interface{}(nil)
	fmt.Println(s3...)
}
