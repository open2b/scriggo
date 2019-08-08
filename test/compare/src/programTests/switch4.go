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
		fmt.Print("case 2")
		a = 20
		fallthrough
	case 3:
		fmt.Print("case 3")
		a = 30
	}
	fmt.Print(a)
}
