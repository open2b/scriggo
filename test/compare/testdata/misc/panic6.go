// paniccheck

package main

import "runtime"

func main() {
	runtime.GOMAXPROCS(1)
	go panic(1)
	runtime.Gosched()
}
