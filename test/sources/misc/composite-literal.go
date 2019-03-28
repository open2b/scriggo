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
	// TODO: stack overflow on x[1]
	//x := m2["k"]
	//_ = x[1]
	//fmt.Println(m2["k"][1], m2["k"]["two"])

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

	_ = map[string]int{"a": 0}
	_ = []int{1, 2, 3}
	_ = [...]int{1, 2, 3}
	_ = [10]int{1, 2, 3}

	_ = []int{0: 1, 2, 10: 3}
	_ = [...]int{0: 1, 2, 20: 3}
	_ = [10]int{0: 1, 2, 2: 3}

	// TODO (Gianluca): add struct tests.
}
