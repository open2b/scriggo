// run

package main

import "fmt"

func p(s ...interface{}) {
	fmt.Printf("%#v\n", s)
}

func main() {
	{
		g := func() int { return 5 }
		f := func(a int) { p(a) }
		f(g())
	}
	{
		g := func() (int, string) { return 5, "b" }
		f := func(a int, b string) { p(a, b) }
		f(g())
	}
	{
		g := func() int { return 5 }
		f := func(a int, b ...string) { p(a, b) }
		f(g())
	}
	{
		g := func() (int, string) { return 5, "b" }
		f := func(a int, b ...string) { p(a, b) }
		f(g())
	}
	{
		g := func() (int, string, string) { return 5, "b", "c" }
		f := func(a int, b ...string) { p(a, b) }
		f(g())
	}
	{
		g := func() (int, string) { return 5, "b" }
		print(g())
	}
	{
		g := func() (int, string) { return 5, "b" }
		println(g())
	}
	{
		h := func() (int, string) { return 5, "b" }
		g := func(a int, b string) (int, string) { return a + 1, b + "c" }
		f := func(a int, b string) { p(a, b) }
		f(g(h()))
	}
}
