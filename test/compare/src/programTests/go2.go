// run

package main

import (
	"fmt"
	"time"
)

func f() {
	fmt.Print("package function")
}

func main() {
	go f()
	time.Sleep(1 * time.Millisecond)
}
