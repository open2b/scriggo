// run

package main

import "fmt"

func main() {
	iface := interface{}(true)
	switch v := iface.(type) {
	default:
		_ = v
		fmt.Print("default")
	case int64:
		_ = v
		fmt.Print("int64")
	}
}