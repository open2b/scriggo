// run

package main

import (
	"fmt"
)

func main() {
	fmt.Print(10)
	fmt.Print(34.51)
	fmt.Print("hello")
	i := interface{}(20)
	fmt.Print(i)
}
