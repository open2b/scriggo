// run

package main

import "fmt"

func main() {
	var chars [6]rune
	chars[0] = 'a'

	var s [10]int
	fmt.Printf("%#v", s)

	var ai [4]int
	ai[0] = ai[1]
	fmt.Println(len(ai))
}
