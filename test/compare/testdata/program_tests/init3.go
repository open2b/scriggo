// run

package main

import (
	"fmt"
)

func init() {
	fmt.Print("init1!")
}

var A = F()

func F() int {
	fmt.Print("F!")
	return 42
}

func main() {
	fmt.Print("main!")
}
