// run

package main

import "fmt"

var a = 1
var b, c = 2, 3
var (
	d    = 4
	e, f = 5, 6
)
var g int = 7
var h, i int = 8, 9
var (
	j    int = 10
	k, l int = 11, 12
)

var m int
var n, o int
var (
	p    int
	q, r int
)

func main() {
	fmt.Println(a, b, c, d, e)
	fmt.Println(f, g, h, i, j)
	fmt.Println(k, l)
	fmt.Println(k, l, m, n, o)
	fmt.Println(p, q, r)
}
