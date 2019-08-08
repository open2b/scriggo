// run

package main

import "fmt"

func main() {
	a := 1
	b := &a
	c := &a
	equal := b == c
	fmt.Println(equal)
}
