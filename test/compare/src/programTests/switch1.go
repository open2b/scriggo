// run

package main

import "fmt"

func main() {
	a := 0
	switch 1 + 1 {
	case 1:
		fmt.Print("case 1")
		a = 10
	case 2:
		fmt.Print("case 1")
		a = 20
	case 3:
		fmt.Print("case 1")
		a = 30
	}
	fmt.Print("a=", a)
	_ = a
}
