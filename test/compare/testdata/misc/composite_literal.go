// run

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
	x := m2["k"]
	_ = x[1]
	fmt.Println(m2["k"][1], m2["k"]["two"])

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

	fmt.Println([]interface{}{} == nil)
	fmt.Println([]byte{} == nil)

	{
		type MyStruct = struct {
			A    int
			B, C float64
		}
		_ = MyStruct{1, 54.3, -43.0}
		_ = MyStruct{A: 1, B: 54.3, C: -43.0}
		_ = MyStruct{A: 1, B: 54.3}
		_ = MyStruct{A: 1, C: -43.0}
	}

	a := [5]int{0, 1, 3: 1, 4: 0}
	fmt.Println(a[0] == 0, a[1] == 1, a[2] == 0, a[3] == 1, a[4] == 0)

	b := []int{2: 1, 4: 0, 0}
	fmt.Println(b[0] == 0, b[1] == 0, b[2] == 1, b[3] == 0, b[4] == 0, b[5] == 0)

	c := []string{"a", "", 3: "", 4: "b"}
	fmt.Println(c[0] == "a", c[1] == "", c[2] == "", c[3] == "", c[4] == "b")

	d := []interface{}{1: nil, 0, nil, 4: []int(nil), 5: "a"}
	fmt.Println(d[0] == nil, d[1] == nil, d[2] == nil, d[3] == nil, d[4] == nil, d[5] == nil)

	e := [3][]int{{}, nil, []int(nil)}
	fmt.Println(e[0] == nil, e[1] == nil, e[2] == nil)

}
