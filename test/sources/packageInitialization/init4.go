//+build ignore

package main

import "fmt"

const A = B
const B = 20

func main() {
	fmt.Println(A)
	fmt.Println(B)
}
