// run

package main

import "fmt"

func f() (r int) {
	func() {
		r = 1
	}()
	return
}

func main() {
	fmt.Println(f())
}
