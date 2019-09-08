// run

package main

import (
	"fmt"
)

func main() {
	a := "hello"
	a = a + " "
	a += "world!"
	fmt.Println(a)
}
