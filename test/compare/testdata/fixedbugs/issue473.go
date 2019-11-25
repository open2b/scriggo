// run

package main

import "fmt"

const (
	a, b = 1, 2
	c, d
)

func main() {
	fmt.Println(a, b, c, d)
	{
		const (
			e, f = 3, 4
			g, h
		)
		fmt.Println(e, f, g, h)
	}
}
