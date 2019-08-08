// run

package main

import "fmt"

func main() {
	a := "hello"
	b := &a
	c := &b
	fmt.Print(**c)
}
