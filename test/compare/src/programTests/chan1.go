// run

package main

import "fmt"

func main() {
	c := make(chan int, 1)
	c <- 10
	fmt.Print(len(c))
	<-c
	fmt.Print(len(c))
}
