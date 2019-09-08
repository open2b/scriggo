// run

package main

import (
	"fmt"
)

func main() {
	fmt.Printf("a")
	for {
		fmt.Printf("b")
		break
		fmt.Printf("???")
	}
	fmt.Printf("c")
}
