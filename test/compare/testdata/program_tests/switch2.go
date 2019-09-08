// run

package main

import "fmt"

func main() {
	a := 0
	switch 10 + 10 {
	case 1:
		fmt.Print("case 1")
		a = 10
	case 2:
		fmt.Print("case 2")
		a = 20
	default:
		fmt.Print("default")
		a = 80
	case 3:
		fmt.Print("case 3")
		a = 30
	}
	fmt.Print(a)
}
