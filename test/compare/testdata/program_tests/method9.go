// run

package main

import (
	"fmt"
	"time"
)

func main() {
	var d time.Duration = 7200000000
	h := time.Duration.Hours(d)
	fmt.Print(h)
}
