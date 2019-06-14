//+build ignore

package main

import (
	"fmt"
)

func main() {
	a := 10
	b := 20

	fmt.Print(a + b)
	fmt.Print(*(&b))
	fmt.Print(a/b, a+b, []int{1, 2, 3}, a-b, "aaa")
	fmt.Print(interface{}("bbb"), 22+4, a+20, b-(a*10))
	fmt.Print()
	c := []string{"a", "b", "c"}
	d := new(int)
	fmt.Print(c, *d)
	fmt.Print(c)
	fmt.Print(*d)
	fmt.Print(*d, *d, 32, *d, c)
	for i := 0; i < 20; i++ {
		fmt.Print(i, i+20, i-20, i*i)
	}
}
