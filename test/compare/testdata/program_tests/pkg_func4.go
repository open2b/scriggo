// run

package main

import "fmt"

func f(a, b, c int) {
	fmt.Println("a, b, c: ", a, b, c)
	return
}

func g(slice []int) {
	fmt.Println(slice, "has len", len(slice))
	return
}

func main() {
	a := 3
	b := 2
	c := 5
	f(a, (b*2)+a-a, c)
	g([]int{a, b, c})
}
