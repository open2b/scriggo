// run

package main

import "fmt"

func main() {

	// Prepare the channel and write a value on it.
	ch := make(chan int, 2)
	ch <- 42

	fmt.Printf("'ch' has len %d\n", len(ch))

	select {
	case <-ch:
		fmt.Println("read from channel 'ch'")
	}

	fmt.Printf("'ch' has len %d\n", len(ch))
}
