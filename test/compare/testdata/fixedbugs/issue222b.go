// run

package main

import "fmt"

type S struct {
	A int
}

func NewS() *S {
	fmt.Println("Called NewS()")
	return &S{}
}

func main() {
	NewS().A++
}
