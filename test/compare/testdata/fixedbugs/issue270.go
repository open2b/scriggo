// run

package main

import "fmt"

var S = struct {
	F int
}{}

func main() {
	fmt.Println(S)
	S.F = 42
	fmt.Println(S)
}
