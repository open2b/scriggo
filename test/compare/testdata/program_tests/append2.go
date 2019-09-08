// run

package main

import "fmt"

func main() {
	for i := range []int{1, 2} {
		s := []int(nil)
		s = append(s, 1)
		fmt.Printf("iteration #%d: s = %v\n", i, s)
	}
}
