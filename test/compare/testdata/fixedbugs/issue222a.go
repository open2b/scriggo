// run

package main

import (
	"fmt"
)

func i() int {
	fmt.Print("i()")
	return 0
}

func main() {
	s := []int{1, 2, 3}
	s[i()] += 5
	fmt.Println(s)
}
