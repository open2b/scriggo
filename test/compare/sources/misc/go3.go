// runcompare

package main

import (
	"fmt"
	"strconv"
	"time"
)

func main() {
	for i := 0; i < 10; i++ {
		go fmt.Print(strconv.Itoa(i))
		time.Sleep(time.Millisecond)
	}
}
