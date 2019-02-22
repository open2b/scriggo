//+build ignore

package main

import "fmt"

const a = 1
const b, c = 2, 3
const (
	d    = 4
	e, f = 5, 6
)
const g int = 7
const h, i int = 8, 9
const (
	j    int = 10
	k, l int = 11, 12
)

func main() {
	fmt.Println(a, b, c, d, e)
	fmt.Println(f, g, h, i, j)
	fmt.Println(k, l)
}
