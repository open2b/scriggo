// run

package main

import (
	"fmt"
	"time"
)

func main() {
	d, _ := time.ParseDuration("20m42s")
	s := fmt.Stringer(d)
	m := s.String
	ss := m()
	fmt.Print("s.String() is ", ss)
}
