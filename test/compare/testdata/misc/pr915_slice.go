// run

package main

import "fmt"

var S = []int{1, 2, 3}

func main() {
	fmt.Println(S)
	S[2] = 441
	fmt.Println(S)
	S[1]--
	fmt.Println(S)
}
