//+build !cgo windows

// paniccheck

package main

func main() {
	ch := make(chan int)
	select {
	case <-ch:
	}
}
