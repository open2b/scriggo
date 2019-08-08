// run

package main

import (
	"fmt"
	"time"
)

func main() {
	var d time.Duration
	d += 7200000000000
	mv := d.Hours
	s := mv()
	fmt.Print(s)
}
