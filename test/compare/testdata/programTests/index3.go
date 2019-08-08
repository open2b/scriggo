// run

package main

import "fmt"

func main() {
	var s1 []int
	var s2 []string

	s1 = []int{1, 2, 3}
	s1[0] = 2

	s2 = []string{"a", "b", "c", "d"}
	s2[2] = "d"

	fmt.Println(s1)
	fmt.Println(s2)
	return
}
