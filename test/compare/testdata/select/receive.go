// run

package main

import "fmt"

func test1() {
	fmt.Println("test1")
	ch := make(chan int, 2)
	ch <- 42
	fmt.Printf("'ch' has len %d\n", len(ch))
	select {
	case <-ch:
		fmt.Println("read from channel 'ch'")
	}
	fmt.Printf("'ch' has len %d\n", len(ch))
}

func test2() {
	fmt.Println("test2")
	ch := make(chan int, 1)
	ch <- 42
	fmt.Println("ch has length", len(ch))
	select {
	case v := <-ch:
		fmt.Println("read", v, "from ch")
	}
	fmt.Println("ch has length", len(ch))
}

func test3() {
	fmt.Println("test3")
	ch := make(chan int, 1)
	ch <- 42
	fmt.Println("ch has length", len(ch))
	select {
	case v, ok := <-ch:
		fmt.Println("read", v, "from ch (ok =", ok, ")")
	}
	fmt.Println("ch has length", len(ch))
}

func test4() {
	ch := make(chan int, 1)
	close(ch)
	fmt.Println("ch has length", len(ch))
	select {
	case v, ok := <-ch:
		fmt.Println("read", v, "from ch (ok =", ok, ")")
	}
	fmt.Println("ch has length", len(ch))
}

func main() {
	test1()
	test2()
	test3()
}
