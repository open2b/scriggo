// run

package main

import (
	"fmt"
)

func main() {
	ch := make(chan int, 10)
	ch <- 42
	fmt.Print(<-ch)
}
