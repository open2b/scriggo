// run

package main

import (
	"fmt"
)

func main() {
	s1 := []int{10, 20, 30}
	s2 := append(s1, 40)
	s3 := append(s1, 50)
	fmt.Print(s1, s2, s3)
}
