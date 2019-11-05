// compile

package main

type A1 [3]int
type A2 [5]A1
type A3 [20][]A2

type Int int

func main() {
	var _ [2]Int
	var _ [2]map[Int]int
}
