// run

package main

import (
	"fmt"
	"time"
)

func addInt(c chan int) {
	time.Sleep(time.Millisecond)
	c <- 42
}

func main() {
	ch := make(chan int, 10)
	go addInt(ch)
	fmt.Print("waiting")
	fmt.Print(<-ch)
}
