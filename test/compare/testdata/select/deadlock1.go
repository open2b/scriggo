// skip : this test waits forever instead of panicking.

package main

func main() {
	ch := make(chan int)
	select {
	case <-ch:
	}
}
