// run

package main

import "fmt"

func test1() {
	fmt.Println("test1")
	select {
	default:
		println("default")
	}
}

func test2() {
	ch := make(chan int, 2)
	select {
	case <-ch:
		panic("case <- ch")
	default:
		fmt.Println("default")
	}
}

func main() {
	test1()
	test2()
}
