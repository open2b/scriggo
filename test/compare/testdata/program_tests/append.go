// run

package main

import (
	"fmt"
)

func main() {

	s := []int{}
	fmt.Print(s)
	s = append(s, []int{1, 2, 3}...)
	fmt.Print(s)
	ns := []int{4, 5, 6}
	s = append(s, ns...)
	fmt.Print(s)
	s = append(s, 7, 8, 9, 10)
	fmt.Print(s)

	b := []byte{}
	fmt.Print(b)
	b = append(b, 1)
	fmt.Print(b)
	b = append(b, 2, 3, 4)
	fmt.Print(b)
	b = append(b, []byte{5, 6, 7, 8}...)
	fmt.Print(b)

	r := []rune{'a'}
	fmt.Print(r)
	r = append(r, []rune{'b', 'c'}...)
	fmt.Print(r)
	r = append(r, 'd', 'e', 'f')
	fmt.Print(r)

	type T struct{}
	t := []T{}
	fmt.Print(len(t))
	t = append(t, T{}, T{})
	fmt.Print(len(t))
	t = append(t, []T{{}, {}}...)
	fmt.Print(len(t))
}
