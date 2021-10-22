// run

package main

import "fmt"

var M = map[int]int{}

func main() {
	fmt.Println(M)
	M[2] = 2
	fmt.Println(M)
	M[1]--
	fmt.Println(M)
}
