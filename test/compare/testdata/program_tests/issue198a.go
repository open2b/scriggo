// run

package main

import "fmt"

func main() {
	s1 := []string{"a", "b", "c"}
	s2 := []string{"c", "d"}
	a := [][]string{s1, s2}
	fmt.Print(a)
}
