//+build !cgo

// paniccheck

package main

func main() {
	ch := make(chan int)
	select {
	case <-ch:
	}
}
