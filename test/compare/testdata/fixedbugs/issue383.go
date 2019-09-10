// run

package main

import (
	"fmt"
)

func main() {

	Println := fmt.Println

	Println("Simple slice expressions")
	{
		a := [4]int{4, 5, 6, 7}
		pa := &a
		s := pa[3:4]
		Println(s, len(s), cap(s))
		Println(pa[:])
		Println((*pa)[1:2])
	}

	Println("Full slice expressions")
	{
		a := [4]int{4, 5, 6, 7}
		pa := &a
		s := pa[3:4:4]
		Println(s, len(s), cap(s))
		Println(pa[:2:3])
		Println((*pa)[1:2:4])
	}

	Println("Selectors")
	{
		type T = struct{ F int }
		t := T{42}
		pt := &t
		Println(pt.F)
		f := pt.F
		Println(f)
		Println(pt.F * (*pt).F)
	}

	Println("Index expressions")
	{
		a := [4]int{4, 5, 6, 7}
		pa := &a
		s := pa[3]
		Println(s)
		Println(pa[1])
		Println((*pa)[0])
	}

	Println("Method calls")
	{
		// TODO:
		// b := strings.Builder{}
		// (&b).WriteString("content")
		// Println((&b).Len())
		// Println(b.Len())
		// Println(b.Cap())
		// b.Grow(5)
		// b.Cap()
		// b.WriteString("aa")
		// Println(b.String())
		// b.WriteString("bb")
		// Println(b.String())
	}

	Println("Method values (1)")
	{
		// TODO.
	}

	Println("Method values (2)")
	{
		// TODO.
	}
}
