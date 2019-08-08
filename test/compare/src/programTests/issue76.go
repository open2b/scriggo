// run

package main

import "fmt"

func typeof(v interface{}) string {
	return fmt.Sprintf("%T", v)
}

func main() {
	{
		var a []int
		var b []byte

		a = []int{}
		b = []byte{}

		fmt.Printf("%T", a)
		fmt.Printf("%T", b)
	}
	{
		var a []int
		var b []byte

		a = []int{}
		b = []byte{}

		fmt.Println(a, typeof(a))
		fmt.Println(b, typeof(b))
	}
}
