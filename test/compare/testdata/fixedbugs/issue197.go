// run

package main

import (
	"fmt"
)

var a = 4382

func main() {

	// Variable (package).
	{
		pa1 := &a
		pa2 := &a
		fmt.Println(pa1 == pa2)
	}

	// Variable (local).
	{
		A := 10
		var (
			ptrA  = &A
			ptrA2 = &(*(ptrA))
		)
		fmt.Println(ptrA == ptrA2)
	}

	// Slice.
	{
		s := []int{1, 2, 3}
		p1 := &s[0]
		p2 := &s[0]
		fmt.Println(p1 == p2)
	}

	// Arrays.
	{
		a := [5]int{1, 2, 3}
		p1 := &a[0]
		p2 := &a[0]
		fmt.Println(p1 == p2)
	}

	// Struct.
	{
		s := struct{ A int }{A: 20}
		p1 := &s.A
		p2 := &s.A
		fmt.Println(p1 == p2)
	}
	{
		s := struct{ A int }{A: 10}
		p1 := &s.A
		fmt.Println(s)
		(*p1) = 42
		fmt.Println(s)
	}
}
