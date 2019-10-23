// skip : needs some synchronization mechanism. https://github.com/open2b/scriggo/issues/420

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
