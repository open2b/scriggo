// run

package main

import (
	"fmt"
)

var m = map[string]int{"20": 12}
var notOk = !ok
var doubleValue = value * 2
var value, ok = m[k()]

func k() string {
	a := 20
	return fmt.Sprintf("%d", a)
}

func main() {
	fmt.Println(m)
	fmt.Println(notOk)
	fmt.Println(doubleValue)
}
