// run

package main

import (
	"fmt"
)

func main() {
	var s = []int{1, 2, 3, 4, 5}
	slices := [][]int{
		s[:],
		s[:0],
		s[0:0],
		s[:3],
		s[0:3],
		s[0:5],
		s[2:5],
		s[2:4],
		s[2:2],
		s[0:0:0],
		s[0:1:2],
		s[2:3:5],
		s[5:5:5],
	}
	for _, slice := range slices {
		fmt.Print(slice, len(slice), cap(slice), "\t")
	}
}
