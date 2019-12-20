// run

package main

import "fmt"

func f() interface{} {
	fmt.Println("f()")
	return 42
}

func main() {
	switch f().(type) {
	case int:
	default:
	}
}