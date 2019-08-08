// run

package main

import (
	"fmt"
)

func main() {
	for _, v := range []int{1, 2, 3} {
		fmt.Print(v, ":cont?")
		if v == 2 {
			fmt.Print("yes!")
			continue
			panic("?")
		}
		fmt.Print("no!")
	}
}
