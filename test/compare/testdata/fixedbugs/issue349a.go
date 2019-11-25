// run

package main

func main() {
	ch := make(chan interface{}, 10)
	ch <- 3
	_ = (<-ch).(int)
}
