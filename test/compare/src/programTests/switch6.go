// run

package main

import "fmt"

func main() {
	fmt.Print("switch: ")
	switch {
	case true:
		fmt.Print("true")
	case false:
		fmt.Print("false")
	}
	fmt.Print(".")
}
