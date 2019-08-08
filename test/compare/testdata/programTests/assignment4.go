// run

package main

import (
	"fmt"
)

func main() {
	s := [][]int{[]int{0, 1}, []int{2, 3}}
	fmt.Print(s, ", ")
	s[0][0] = 5
	fmt.Print(s)
}
