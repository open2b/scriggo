// run

package main

import "fmt"

func main() {
	count := 0
	if t := 1; false {
		count = count + 1
		_ = t
		t := 7
		_ = t
	} else {
		count = count - t
	}
	fmt.Print(count)
}
