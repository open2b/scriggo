// run

package main

import "fmt"

func main() {

	a := "a"
	b := "b"
	c := "c"

	pb := &b
	*pb = "newb"

	fmt.Print(a, b, c)
}
