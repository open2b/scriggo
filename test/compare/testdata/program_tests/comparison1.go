// run

package main

import "fmt"

func main() {
	var a interface{}
	if a == nil {
		fmt.Print("is nil")
	} else {
		fmt.Print("is not nil")
	}
}
