// run

package main

func main() {
	var ss [][]int

	ss = [][]int{
		[]int{10, 20},
		[]int{},
		[]int{30, 40, 50},
	}
	ss[1] = []int{25, 26}

	_ = ss
	return
}
