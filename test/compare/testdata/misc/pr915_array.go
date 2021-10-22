// run

package main

import "fmt"

var A [3]int

func main() {
	fmt.Println(A)
	A[2] = 32
	fmt.Println(A)
	A[1]++
	fmt.Println(A)
}
