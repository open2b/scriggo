// run

package main

import (
	"fmt"
)

func main() {

	// Builtin 'close'.

	ch := make(chan int)
	defer close(ch)
	defer close(make(chan int))
	ch2 := make(chan int)
	go close(ch2)
	go close(make(chan string))

	// Builtin 'copy'.

	src := []byte{4, 5, 6}
	dst := make([]byte, len(src))
	defer fmt.Printf("%v\n", dst)
	defer copy(dst, src)
	defer fmt.Printf("%v\n", dst)
	copyDone := make(chan struct{})
	go func() {
		copy(dst, "123")
		close(copyDone)
	}()
	<-copyDone

	// Builtin 'delete'.

	m := map[string]string{"a": "A", "b": "B"}
	defer fmt.Println(m["a"])
	defer delete(m, "a")
	n := map[int]int{1: 2, 2: 3}
	defer fmt.Println(n[2])
	deleteDone := make(chan struct{})
	go func() {
		delete(n, 2)
		close(deleteDone)
	}()
	<-deleteDone

	// Builtin 'print'.

	defer print("print")
	defer print(1, 2, 3)
	defer print()

	// Builtin 'println'.

	defer println("println")
	defer println(4, 5, 6)
	defer println()
}
