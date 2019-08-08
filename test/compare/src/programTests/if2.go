// run

package main

import "fmt"

func main() {
	a := 10
	c := 0
	if a <= 5 {
		c = 10
		fmt.Print("then")
	} else {
		c = 20
		fmt.Print("else")
	}
	fmt.Print(c)
}
