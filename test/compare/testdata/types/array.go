// run

package main

type Int int

type A1 [3]Int
type A2 [56]A1

func main() {
	var _ A1
	var _ A2
	var _ [5]A1
	var _ [20][]A2
	var _ [2]Int
	var _ [2]map[Int]int
	var _ [43][][][][][]int
	var _ [43][][][][][]Int
}
