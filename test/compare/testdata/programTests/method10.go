// run

package main

import (
	"fmt"
	"time"
)

func main() {
	var d time.Duration = 7200000000
	expr := time.Duration.Hours
	h := expr(d)
	fmt.Print(h)
}
