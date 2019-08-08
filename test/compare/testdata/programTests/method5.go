// run

package main

import (
	"fmt"
	"time"
)

func main() {
	var d time.Duration
	d += 7200000000000
	s := d.Hours()
	fmt.Print(s)
}
