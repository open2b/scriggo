// run

package main

import "fmt"

func main() {
	c := make(chan int)
	fmt.Print("closing: ")
	close(c)
	fmt.Print("ok")
}
