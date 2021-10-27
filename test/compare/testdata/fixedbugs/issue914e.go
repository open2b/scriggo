// run

package main

import "fmt"

var x [1]int

func main() {
	x = [1]int{1}
	y := x
	fmt.Println(x, y)
	x = [1]int{2}
	fmt.Println(x, y)
}
