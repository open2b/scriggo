// run

package main

import (
	"fmt"
)

func main() {
	for {
		fmt.Print("a")
		break
	}
	for i := 0; ; i = i + 2 {
		fmt.Print("b")
		if i > 10 {
			break
		}
	}
	for {
		fmt.Print("c1")
		for {
			fmt.Print("c2")
			break
			panic("")
		}
		break
		panic("")
	}
}
