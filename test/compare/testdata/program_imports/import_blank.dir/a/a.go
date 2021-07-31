package a

import "fmt"

var A int

func init() {
	fmt.Println("a: init!")
	A = 42
}
