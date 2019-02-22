//+build ignore

package main

import (
	"fmt"
)

func main() {
	m1 := map[string]interface{}{}
	m1["k"] = []int{1, 2, 3}
	fmt.Println(m1)

	m2 := map[string]map[interface{}]string{}
	m2["k"] = map[interface{}]string{
		1:     "v1",
		"two": "v2",
	}
	fmt.Println(m2)

	s1 := []int{1, 2, 3}
	s1 = append(s1, s1[0], s1[1])
	fmt.Println(s1)

	s2 := [][]int{}
	s2 = append(s2, []int{1, 2})
	s2 = append(s2, []int{3, 4})
	fmt.Println(s2)
	sum := 0
	for x := range s2 {
		for y := range s2[x] {
			sum += s2[x][y]
		}
	}
	fmt.Println(sum)
}
