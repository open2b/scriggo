//+build

package main

import "fmt"

func main() {
	a, b := 10, 20
	fmt.Println(a, b)
	a, b = b, a
	fmt.Println(a, b)

	x := []int{1, 2, 3}
	i := 0
	i, x[i] = 1, 2

	i = 0
	x[i], i = 2, 1

	x[0], x[0] = 1, 2

	x[1], x[2] = 4, 5

	v := 10
	fmt.Println(v)
	ref := &v
	*ref = 2
	fmt.Println(v)

	i = 2
	x = []int{3, 5, 7}
	for i, x[i] = range x {
		break
	}
	fmt.Println(i, x)
}
