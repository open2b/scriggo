// run

package main

import "fmt"

func main() {
	a := 1
	b := &a
	c := &a
	fmt.Println(b == c)
}
