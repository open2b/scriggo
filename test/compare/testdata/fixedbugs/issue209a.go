// run

package main

import "fmt"

const (
	abit, amask = 1 << iota, 1<<iota - 1
	bbit, bmask = 1 << iota, 1<<iota - 1
	cbit, cmask = 1 << iota, 1<<iota - 1
)

func main() {
	fmt.Print(abit, amask, bbit, bmask, cbit, cmask)
}
