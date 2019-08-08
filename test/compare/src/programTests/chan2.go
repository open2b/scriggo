// run

package main

import "fmt"

func main() {
	ch := make(chan int, 4)
	ch <- 5
	v := <-ch
	fmt.Print(v)
}
