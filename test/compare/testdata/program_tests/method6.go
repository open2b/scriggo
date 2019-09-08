// run

package main

import (
	"fmt"
	"time"
)

func main() {
	d, _ := time.ParseDuration("10m")
	s := fmt.Stringer(d)
	ss := s.String()
	fmt.Print("s.String() is ", ss)
}
