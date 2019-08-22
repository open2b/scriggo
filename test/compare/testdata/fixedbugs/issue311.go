// run

package main

import (
	"fmt"
)

func array() {
	a := [3][]int{
		[]int{1, 2, 3},
		nil,
		[]int{},
	}
	fmt.Println("array:", a)
}

func slice() {
	s := [][]string{
		[]string{"a", "b"},
		nil,
		[]string{},
	}
	fmt.Println("slice:", s)
}

func mapKey() {
	m := map[interface{}]int{
		6:                10,
		"x":              43,
		nil:              11,
		interface{}(nil): 1,
	}
	fmt.Println("mapKey:", m)
}

func mapValue() {
	m := map[int]interface{}{
		6:  10,
		43: "x",
		11: nil,
		1:  []int(nil),
	}
	fmt.Println("mapValue:", m)
}

func structImplicit() {
	s := struct {
		A, B []int
	}{
		[]int{1, 2, 3}, nil,
	}
	fmt.Println("structImplicit:", s)
}

func structExplicit() {
	s := struct {
		A, B []int
	}{
		A: []int{1, 2, 3},
		B: nil,
	}
	fmt.Println("structImplicit:", s)
}

func main() {
	array()
	slice()
	mapKey()
	mapValue()
	structImplicit()
	structExplicit()
}
