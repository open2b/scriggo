// run

package main

import (
	"fmt"
)

func main() {
	const d = 20.0
	fmt.Printf("%T", d)
	i := interface{}(d)
	fmt.Printf("%T", d)
	fmt.Print(i)
}
