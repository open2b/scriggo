// +build ignore

package main

import (
	"fmt"
)

func main() {

	{
		a := 10
		switch a + 1 {
		case 11:
			fmt.Println(11)
		}
	}
	{
		switch false {
		}
	}
	{
		switch interface{}(4).(type) {
		case int:
			fmt.Println("is an int!")
		}
	}
	{
		switch 4 {
		default:
			fmt.Println("unknown")
		case 2:
			fmt.Println("is 2")
		case 3:
			fmt.Println("is 3")
		case 4:
			fmt.Println("is 4")
		}
	}
	{
		unhex := func(c byte) byte {
			switch {
			case '0' <= c && c <= '9':
				return c - '0'
			case 'a' <= c && c <= 'f':
				return c - 'a' + 10
			case 'A' <= c && c <= 'F':
				return c - 'A' + 10
			}
			return 0
		}
		fmt.Println(unhex('0'))
		fmt.Println(unhex('a'))
		fmt.Println(unhex('F'))
		fmt.Println(unhex('9'))
	}
	{
		v := 42
		switch v {
		case 100:
			fmt.Println(100)
			fallthrough
		case 42:
			fmt.Println(42)
			fallthrough
		case 1:
			fmt.Println(1)
			fallthrough
		default:
			fmt.Println("default")
		}
	}
	{
		typeName := func(v interface{}) string {
			switch v.(type) {
			case int:
				return "int"
			case string:
				return "string"
			default:
				return "unknown"
			}
		}
		fmt.Println(typeName("a"))
		fmt.Println(typeName(3))
		fmt.Println(typeName([]int{1, 2, 3}))
	}
	{
		v := interface{}(3)
		switch u := v.(type) {
		default:
			fmt.Println(u)
		}
	}
	{
		do := func(v interface{}) int {
			switch u := v.(type) {
			case int:
				return u * 2
			case string:
				return len(u)
			}
			return 3
		}
		fmt.Println(do(21))
		fmt.Println(do("bitrab"))
		fmt.Println(do(3.142))
	}
	{
		pluralEnding := func(n int) string {
			ending := ""
			switch n {
			case 1:
			default:
				ending = "s"
			}
			return ending
		}
		fmt.Println("foo%s\n", pluralEnding(1))
		fmt.Println("bar%s\n", pluralEnding(2))
	}
}
