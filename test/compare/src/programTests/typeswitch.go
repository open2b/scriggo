// run

package main

import "fmt"

func main() {
	i := interface{}(int64(5))
	v := 0
	switch i.(type) {
	case string:
		v = 10
	case int64:
		v = 20
	default:
		v = 30
	}

	fmt.Print(v)
	return
}
