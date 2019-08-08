// run

package main

import "fmt"

func main() {
	b := map[interface{}]interface{}{
		nil: true,
	}
	if a, ok := b[nil]; ok {
		_ = a
		fmt.Print("ok")
	} else {
		fmt.Print("no")
	}
}
