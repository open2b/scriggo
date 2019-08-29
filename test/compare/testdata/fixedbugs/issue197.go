// run

package main

import (
	"fmt"
)

var a = 4382

func main() {
	{
		pa1 := &a
		pa2 := &a
		fmt.Println(pa1 == pa2)
	}
	{
		A := 10
		var (
			ptrA  = &A
			ptrA2 = &(*(ptrA))
		)
		fmt.Println(ptrA == ptrA2)
	}
	{
		s := []int{1, 2, 3}
		p1 := &s[0]
		p2 := &s[0]
		fmt.Println(p1 == p2)
	}
	{
		a := [5]int{1, 2, 3}
		p1 := &a[0]
		p2 := &a[0]
		fmt.Println(p1 == p2)
	}
	{
		s := struct{ A int }{A: 20}
		p1 := &s.A
		p2 := &s.A
		fmt.Println(p1 == p2)
	}
}
