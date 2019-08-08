// skip : cannot pass the argument

// run

package main

import "fmt"

func main() {
	defer func(s string) {
		fmt.Print(s)
	}("b")
	fmt.Print("a")
}
