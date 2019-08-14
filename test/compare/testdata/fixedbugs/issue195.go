// run

package main

import (
	"fmt"
)

func pointer() *int {
	a := 10
	return &a
}

func main() {
	fmt.Print(*pointer())
}
