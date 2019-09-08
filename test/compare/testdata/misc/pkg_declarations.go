// run

package main

import "fmt"

var A = "package variable"

func main() {
	A := "local variable"
	fmt.Println(A)
}
