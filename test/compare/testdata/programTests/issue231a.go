// skip

package main

import (
	"fmt"
	"time"
)

func f(s string, i int, f float64, si []int) {
	fmt.Print(s, i, f, si)
}

func main() {
	go f("str", 42, -45.120, []int{3, 4, 5})
	time.Sleep(3 * time.Millisecond)
}
