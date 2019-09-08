// run

package main

import (
	"fmt"
)

func main() {
	s := []int{}
	fmt.Print(s)
	s = append(s, []int{1, 2, 3}...)
	fmt.Print(s)
	ns := []int{4, 5, 6}
	s = append(s, ns...)
	fmt.Print(s)
}
