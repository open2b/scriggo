// run

package main

import "fmt"

func main() {
	var s = make([]func() int, 1)
	s[0] = func() int { return 10 }
	res := s[0]()
	fmt.Println(res)
}
