// run

package main

import "fmt"

type SliceInt = []int

func main() {
	si := SliceInt{10, 20, 30}
	fmt.Print(len(si))
	fmt.Print(si)
}
