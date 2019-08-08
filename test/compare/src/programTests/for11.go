// run

package main

import (
	"fmt"
)

func main() {
	for i := 0; i < 5; i++ {
		fmt.Print(i)
	}
	fmt.Print(",")
	for i := 0; i < 5; {
		fmt.Print(i)
		i++
	}
	fmt.Print(",")
	for i := 0; i < 5; {
		i++
		fmt.Print(i)
	}
	fmt.Print(",")
	i := 0
	for ; i < 5; i++ {
		fmt.Print(i)
	}
	fmt.Print(",")
	for i := 0; i < 5; i++ {
		i++
		fmt.Print(i)
	}
}
