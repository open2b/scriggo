// run

package main

import (
	"fmt"
	"time"
)

func main() {
	d, _ := time.ParseDuration("20m42s")
	s := fmt.Stringer(d)
	m := fmt.Stringer.String
	ss := m(s)
	fmt.Print("s.String() is ", ss)
}
