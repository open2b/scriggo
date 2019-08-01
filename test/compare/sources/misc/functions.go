// run

package main

import "fmt"

func MakeSlice1(a, b, c int) []int {
	return []int{a, b, c}
}

func MakeMap1(k1, v1 int) map[int]int {
	return map[int]int{k1: v1}
}

func MakeInterface() interface{} {
	return 0
}

func MakeSlice2(args ...int) []int {
	slice := make([]int, len(args))
	for i, a := range args {
		slice[i] = a
	}
	return slice
}

func MakeFunc1() func() {
	return func() { fmt.Println("message") }
}

func main() {
	MakeFunc1()()

	fmt.Println(MakeMap1(1, 10))
	fmt.Println(MakeInterface())
}
