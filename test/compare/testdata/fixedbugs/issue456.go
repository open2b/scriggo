// run

package main

func main() {

	ch1 := make(chan<- int)
	ch2 := make(<-chan int)

	select {
	case ch1 <- 1:
	case <-ch2:
	default:
	}

}
