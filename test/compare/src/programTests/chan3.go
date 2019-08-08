// run

package main

import "fmt"

func main() {
	ch := make(chan int, 4)
	ch <- 42
	v, ok := <-ch
	fmt.Print(v, ok)
}
