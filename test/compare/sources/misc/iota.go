// run

package main

import "fmt"

const c1 = iota

const (
	c2 = iota
	c3 = iota
)
const (
	c4 = iota
	c5
)
const (
	c6 = (iota + 1) * 20
	c7 = iota * 30
	c8
)

const (
	c9  float64 = (iota * 3) + iota
	c10 float32 = (iota * 3) + iota
	c11
	c12
)

func main() {
	fmt.Print(c1, c2, c3, c4, c5, c6, c7, c8, c9, c10, c11, c12)
	{
		const c1 = iota
		const (
			c2 = iota
			c3 = iota
		)
		const (
			c4 = iota
			c5
		)
		const (
			c6 = (iota + 1) * 20
			c7 = iota * 30
			c8
		)
		fmt.Print(c1, c2, c3, c4, c5, c6, c7, c8)
	}
}
