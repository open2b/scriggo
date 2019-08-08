// run

package main

import (
	"fmt"
	"time"
)

func main() {
	go func() {
		fmt.Print("func literal")
	}()
	time.Sleep(1 * time.Millisecond)
}
