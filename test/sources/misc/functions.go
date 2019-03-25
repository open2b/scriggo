//+build ignore

package main

import "fmt"

func MakeSlice1(a, b, c int) []int {
	return []int{a, b, c}
}

func MakeMap1(k1, v1 int) map[int]int {
	return map[int]int{k1: v1}
}

// TODO (Gianluca): «10:26: invalid argument args (type int) for len»
// func MakeSlice2(args ...int) []int {
// 	slice := make([]int, len(args))
// 	for i, a := range args {
// 		slice[i] = a
// 	}
// 	return slice
// }

// TODO (Gianluca): not implemented, wait for type-checker to return types as constants.
// func MakeFunc1() func() {
// 	return func() { fmt.Println("message") }
// }

func main() {
	// TODO (Gianluca): not implemented, wait for type-checker to return types as constants.
	// MakeFunc1()()

	fmt.Println(MakeMap1(1, 10))
}
