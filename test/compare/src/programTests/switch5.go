// run

package main

import "fmt"

func main() {
	a := 6
	b := 7
	switch 42 {
	case a * b:
		fmt.Print("a * b")
	default:
		fmt.Print("default")
	}
}
