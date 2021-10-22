// run

package main

import "fmt"

func main() {
	var S struct{ F int }
	fmt.Println(S)
	func() {
		S.F = 2
	}()
	fmt.Println(S)
}
