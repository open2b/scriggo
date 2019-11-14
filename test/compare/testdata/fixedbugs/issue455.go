// run

package main

func main() {

	ch1 := make(chan int)
	select {
	case <-ch1:
	case ch1 <- 1:
	default:
	}

	ch2 := make(<-chan int)
	select {
	case <-ch2:
	default:
	}

	ch3 := make(chan<- int)
	select {
	case ch3 <- 1:
	default:
	}

}
