// run

package main

import (
	"fmt"
)

func arg() int {
	fmt.Print("arg")
	return 1
}

func f(int) {}

func main() {
	fmt.Print("pre, ")
	go f(arg())
	fmt.Print(", post")
}
