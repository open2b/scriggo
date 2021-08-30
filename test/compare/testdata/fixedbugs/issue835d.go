// run

package main

import "fmt"

func main() {
	x := 10
	xp := &x
	x, y := 20, 30
	_, _ = y, y
	fmt.Println(*xp)
}
