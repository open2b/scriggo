// run

package main

import (
	"fmt"
)

func main() {

	// Copy of an array.
	{
		a := [3]int{1, 2, 3}
		b := a
		b[2] = -10
		fmt.Println(a, b)
	}

	// Copy of a struct.
	{
		s1 := struct{ A int }{42}
		s2 := s1
		s1.A = 80
		fmt.Println(s1, s2)
	}
}
