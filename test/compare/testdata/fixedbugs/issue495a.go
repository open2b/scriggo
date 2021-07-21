// run

package main

import (
	"fmt"
)

func g() (_ string) {
	return "hello"
}

func main() {
	x := g()
	fmt.Println("x: ", x)
}
