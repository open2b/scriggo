// run

package main

import "fmt"

func main() {
	a := 0
	switch 2 + 2 {
	case 1:
		fmt.Print("case 1")
		a = 10
	default:
		fmt.Print("default")
		a = 80
		fallthrough
	case 2:
		fmt.Print("case 2")
		a = 1
		fallthrough
	case 3:
		fmt.Print("case 3")
		a = 30
	case 40:
		fmt.Print("case 4")
		a = 3
	}
	fmt.Print(a)
}
