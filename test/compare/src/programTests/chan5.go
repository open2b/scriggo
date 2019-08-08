// run

package main

import "fmt"

func main() {
	ch := make(chan int, 4)
	for i := 0; i < 4; i++ {
		ch <- i
	}
	for i := 0; i < 4; i++ {
		v := <-ch
		fmt.Print(v)
	}
}
