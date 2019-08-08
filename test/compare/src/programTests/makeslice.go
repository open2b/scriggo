// run

package main

import (
	"fmt"
)

func main() {
	var s []int
	var l, c int
	s = make([]int, 10)
	fmt.Printf("%v, len: %d, cap: %d\n", s, len(s), cap(s))
	s = make([]int, 4, 6)
	fmt.Printf("%v, len: %d, cap: %d\n", s, len(s), cap(s))
	s = make([]int, 0, 3)
	fmt.Printf("%v, len: %d, cap: %d\n", s, len(s), cap(s))
	l = 2
	s = make([]int, l)
	fmt.Printf("%v, len: %d, cap: %d\n", s, len(s), cap(s))
	c = 5
	s = make([]int, l, c)
	fmt.Printf("%v, len: %d, cap: %d\n", s, len(s), cap(s))
	s = make([]int, l, 20)
	fmt.Printf("%v, len: %d, cap: %d\n", s, len(s), cap(s))
	s = make([]int, 0, c)
	fmt.Printf("%v, len: %d, cap: %d\n", s, len(s), cap(s))
}
