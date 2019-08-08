// run

package main

import (
	"fmt"
)

func main() {
	s1 := make([]int, 2)
	fmt.Print("s1: ", len(s1), cap(s1))
	s2 := make([]int, 2, 5)
	fmt.Print("/s2: ", len(s2), cap(s2))
}
