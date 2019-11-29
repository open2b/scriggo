// run

package main

import "fmt"

func main() {
	a := 0
	func() {
		b := &a
		*b = 1
		d := &a
		fmt.Println(b == d)
	}()
}
