// skip : panic: interface conversion: interface {} is nil, not *vm.callable

// run

package main

import (
	"fmt"
	"strings"
	"unicode"
)

var F func(int) int

func main() {
	{
		f := func(c rune) bool {
			return unicode.Is(unicode.Han, c)
		}
		fmt.Println(strings.IndexFunc("Hello, 世界\n", f))
		fmt.Println(strings.IndexFunc("Hello, world\n", f))
	}
	{
		var m = map[string]func(int) int{}
		m["inc"] = func(x int) int { return x + 1 }
		fmt.Printf("%T %d\n", m["inc"], m["inc"](3))
	}
	{
		var s = make([]func(int) int, 1)
		s[0] = func(x int) int { return x + 1 }
		fmt.Printf("%T %d\n", s[0], s[0](3))
	}
	{
		var s = make([]interface{}, 1)
		s[0] = func(x int) int { return x + 1 }
		fmt.Printf("%T %d\n", s[0], s[0].(func(x int) int)(3))
	}
	{
		var f interface{} = func(x int) int { return x + 1 }
		if _, ok := f.(func(x int) int); ok {
			fmt.Println("ok")
		}
	}
}
