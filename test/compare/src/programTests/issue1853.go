// run

package main

import "fmt"

func main() {
	var s = make([]func(int) int, 1)
	s[0] = func(x int) int { return x + 1 }
	res := s[0](3)
	fmt.Println(res)
}
