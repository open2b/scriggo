// run

package main

import "fmt"

func main() {
	i := interface{}("message")
	switch v := i.(type) {
	case string:
		fmt.Println("message is", v)
	default:
		_ = v
		panic("")
	}
}
