// run

package main

import "fmt"

func main() {
	b := map[interface{}]interface{}{
		nil: true,
	}
	fmt.Print(b)
}
