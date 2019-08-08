// run

package main

import (
	"fmt"
)

func main() {
	s := []int{}
	fmt.Print(s)
	s = append(s, 1)
	fmt.Print(s)
	s = append(s, 2, 3)
	fmt.Print(s)
	s = append(s)
	fmt.Print(s)
}
