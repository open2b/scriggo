// run

package main

import (
	"fmt"
)

func main() {

	ch := make(chan int, 1)
	fmt.Println("ch has length", len(ch))

	select {
	case ch <- 20:
		fmt.Println("written 20 on ch")
	}

	fmt.Println("ch has length", len(ch))
}
