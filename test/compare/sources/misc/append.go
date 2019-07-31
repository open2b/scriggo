// runcmp

package main

import "fmt"

func printS(s []int) {
	fmt.Printf("[len=%d] -> %v\n", len(s), s)
}

func main() {
	var s []int
	printS(s)
	s = append(s, 0)
	printS(s)
	s = append(s, 5)
	printS(s)
	s = append(s, 10, 20, 30)
	printS(s)
}
