// run

package main

import "fmt"

func main() {
	p := fmt.Println
	{
		a := 2
		p(a)
	}
	{
		a, b := 10, 20
		p(a, b)
	}
	{
		a, b := 20, "string"
		p(a, b)
	}
	{
		a := []int{1, 2, 3}
		p(a)
	}
	{
		a := [][]string{
			{"a", "b", "c"},
			{"d", "e", "f"},
		}
		p(a)
	}
	{
		a := [][]int{}
		p(a)
	}
	{
		a, b := 10, 20
		p(a, b)
		a, b = b, a
		p(a, b)
	}
	{
		a := []int{1, 2, 3}
		p(a)
		a[0] = 20
		p(a)
	}
	{
		s := "str"
		s0 := s[0]
		s1 := s[1]
		p(s, s0, s1)
	}
	{
		f := func() int {
			fmt.Print("called")
			return 0
		}
		_ = f()
	}
	{
		a, b, _ := 10, 20, 30
		p(a, b)
	}

}
