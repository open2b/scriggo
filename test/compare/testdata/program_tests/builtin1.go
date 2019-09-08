// run

package main

import (
	"fmt"
)

func main() {
	s1 := make([]int, 2, 10)
	c1 := cap(s1)
	fmt.Print(c1, ",")
	s2 := []int{1, 2, 3}
	c2 := cap(s2)
	fmt.Print(c2)
}
