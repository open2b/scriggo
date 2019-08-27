//+build !linux

// TODO(Gianluca): remove the build tag

// paniccheck

package main

func main() {
	ch := make(chan int)
	select {
	case <-ch:
	}
}
