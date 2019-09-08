// run

package main

import "fmt"

func main() {
	a := 10
	c := 0
	if a > 5 {
		fmt.Print("then")
		c = 1
	} else {
		fmt.Print("else")
		c = 2
	}
	fmt.Print("c=", c)
}
