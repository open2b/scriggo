// run

package main

import "fmt"

func random(n int) []int8 {
	r := make(chan int8, n)
	for i := 0; i < n; i++ {
		select {
		case r <- 0:
		case r <- 1:
		}
	}
	slice := []int8{}
	for {
		select {
		default:
			return slice
		case elem := <-r:
			slice = append(slice, elem)
		}
	}
}

func main() {
	fmt.Println(len(random(2)))
	fmt.Println(len(random(15)))
}
