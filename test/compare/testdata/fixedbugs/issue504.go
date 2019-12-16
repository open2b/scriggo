// run

package main

import (
	"fmt"
)

func main() {
	matrix := [][]string{{"a", "b"}, {"c", "d"}}
	for _, d := range matrix {
		for _, d := range d {
			fmt.Print(d)
		}
	}
}
