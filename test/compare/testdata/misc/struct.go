// run

package main

import "fmt"

func main() {
	{
		s := struct{}{}
		fmt.Printf("%v %T", s, s)
	}
	{
		s := struct{ A int }{}
		fmt.Printf("%v %T", s, s)
		fmt.Printf("%v %T", s.A, s.A)
	}
	{
		s := struct{ A int }{20}
		fmt.Printf("%v %T", s, s)
		fmt.Printf("%v %T", s.A, s.A)
	}
	{
		s := struct{ A int }{A: 20}
		fmt.Printf("%v %T", s, s)
		fmt.Printf("%v %T", s.A, s.A)
	}
	{
		ps := &struct{}{}
		fmt.Printf("%v %T", ps, ps)
	}
	{
		ps := &struct{ A int }{}
		fmt.Printf("%v %T", ps, ps)
		fmt.Printf("%v %T", ps.A, ps.A)
		fmt.Printf("%v %T", (*ps).A, (*ps).A)
	}
	{
		ps := &struct{ A int }{20}
		fmt.Printf("%v %T", ps, ps)
		fmt.Printf("%v %T", ps.A, ps.A)
		fmt.Printf("%v %T", (*ps).A, (*ps).A)
	}
	{
		ps := &struct{ A int }{A: 20}
		fmt.Printf("%v %T", ps, ps)
		fmt.Printf("%v %T", ps.A, ps.A)
		fmt.Printf("%v %T", (*ps).A, (*ps).A)
	}
	{
		s := struct{ A int }{20}
		fmt.Printf("%v %T", s, s)
		fmt.Printf("%v %T", s.A, s.A)
		s.A = 50
		fmt.Printf("%v %T", s, s)
		fmt.Printf("%v %T", s.A, s.A)
	}
}
