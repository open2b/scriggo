// run

package main

func main() {
	type Int int
	type Int2 Int
	type Int3 Int2
	type SliceInt3 []Int3
	type SliceSliceInt3 []SliceInt3
	var s SliceSliceInt3
	_ = s
}
