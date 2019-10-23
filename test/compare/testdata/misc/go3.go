// skip : needs some synchronization mechanism. https://github.com/open2b/scriggo/issues/420

// run

package main

import (
	"fmt"
	"strconv"
	"time"
)

func main() {
	for i := 0; i < 10; i++ {
		go fmt.Print(strconv.Itoa(i))
		time.Sleep(2 * time.Millisecond)
	}
}
