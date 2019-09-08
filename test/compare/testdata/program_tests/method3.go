// run

package main

import (
	"fmt"
	"time"
)

func main() {
	var d time.Duration
	d += 7200000000000
	dp := &d
	s := dp.Hours()
	fmt.Print(s)
}
