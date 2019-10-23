// skip : needs some synchronization mechanism. https://github.com/open2b/scriggo/issues/420

// run

package main

import (
	"fmt"
	"time"
)

func f() {
	fmt.Print("c")
}

func main() {
	go fmt.Print("a")
	time.Sleep(time.Millisecond)
	fmt.Print("b")
	go f()
	time.Sleep(time.Millisecond)
	fmt.Print("d")
}
