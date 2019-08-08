// run

package main

import (
	"fmt"
)

func a() int {
	fmt.Print("a")
	return 0
}

func b() int {
	fmt.Print("b")
	return 0
}
func c() int {
	fmt.Print("c")
	return 0
}
func d() int {
	fmt.Print("d")
	return 0
}
func e() int {
	fmt.Print("e")
	return 0
}

func main() {
	s := []int{a(), b(), c(), d(), e()}
	_ = s
}
