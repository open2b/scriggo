// run

package main

import (
	"fmt"
)

func test1() {
	ch := make(chan int, 1)
	fmt.Println(len(ch))
	select {
	case ch <- 20:
		fmt.Println("written 20 on ch")
	}
	fmt.Println(len(ch))
}

func test2() {
	fmt.Println("test2")
	ch := make(chan []int, 2)
	select {
	case ch <- []int{1, 2, 3}:
		fmt.Println("slice written to the channel")
	}
	fmt.Println(<-ch)
}

func main() {
	test1()
	test2()
}
