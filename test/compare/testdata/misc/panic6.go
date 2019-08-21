// paniccheck

package main

func main() {
	go panic(1)
	<-make(chan struct{})
}
