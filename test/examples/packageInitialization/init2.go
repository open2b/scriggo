//+build ignore

package main

import "fmt"

var A = B
var B = 10

func main() {
	fmt.Println(A)
	fmt.Println(B)
}
