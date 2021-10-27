// run

package main

import "fmt"

var S = struct{ F int }{}

func main() {
	fmt.Println(S)
	S.F = 441
	fmt.Println(S)
	S.F++
	fmt.Println(S)
	S.F += 32
	fmt.Println(S)
	S.F--
	fmt.Println(S)
}
