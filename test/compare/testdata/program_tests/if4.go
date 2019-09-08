// run

package main

import "fmt"

func main() {
	c := 0
	if x := 1; x == 1 {
		c = 1
		fmt.Print("then")
		fmt.Print("x is ", x)
	} else {
		c = 2
		fmt.Print("else")
	}
	fmt.Print(c)
}
