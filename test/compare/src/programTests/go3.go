// run

package main

import (
	"fmt"
	"time"
)

func main() {
	go fmt.Print("hello")
	time.Sleep(1 * time.Millisecond)
}
