// skip : needs some synchronization mechanism. https://github.com/open2b/scriggo/issues/420

// run

package main

import (
	"fmt"
	"time"
)

func wait() {
	time.Sleep(time.Millisecond)
}

func f(s string) {
	fmt.Print(s)
}

func main() {
	go fmt.Print("a")
	wait()
	fmt.Print("b")
	go f("c")
	wait()
	fmt.Print("d")
	fun := f
	go fun("e")
	wait()
}
