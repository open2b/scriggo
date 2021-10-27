// run

package main

import "fmt"

var x []int

func main() {
	x = []int{1}
	y := x
	fmt.Println(x, y)
	x = []int{2}
	fmt.Println(x, y)
}
