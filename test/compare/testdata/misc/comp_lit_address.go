// run

package main

import (
	"fmt"
)

func main() {

	// Arrays.
	a1 := &[3]int{}
	fmt.Println(a1)

	// Slices.
	s1 := &[]int{}
	fmt.Println(s1)
	s2 := &[]string{}
	fmt.Println(s2)
	s3 := &[]int{1, 2, 3}
	fmt.Println(s3)
	s4 := &[]interface{}{1, 10, "hello", []int{60, 403, 3}}
	fmt.Println(s4)
	s5 := &[]*[]int{&[]int{5, 6, 7}}
	fmt.Println(len(*s5))
	fmt.Println(len(*(*s5)[0]))

	// Maps.
	m1 := &map[string]string{}
	fmt.Println(m1)
	m2 := &map[string]int{"h": 10, "h2": 20}
	fmt.Println(m2)

	// Structs.
	struct1 := &struct{ A, B int }{}
	fmt.Println(struct1)
	struct2 := &struct{ A, B int }{50, 60}
	fmt.Println(struct2)

}
