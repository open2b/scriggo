// run

package main

import "fmt"

func main() {
	ch := make(chan interface{}, 10)
	ch <- 42
	fmt.Println(<-ch)
}
