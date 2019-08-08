// run

package main

import (
	"fmt"
	"time"
)

func main() {
	d, _ := time.ParseDuration("20m42s")
	s := fmt.Stringer(d)
	ss := fmt.Stringer.String(s)
	fmt.Print("s.String() is ", ss)
}
