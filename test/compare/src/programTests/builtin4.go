// run

package main

import (
	"fmt"
)

func main() {
	s1 := []int{1, 2, 3}
	s2 := []int{4, 5, 6}
	s3 := append(s1, s2...)
	fmt.Print(s1, s2, s3)
}
