// run

package main

import "fmt"

func f() (int, int) {
	return 1, 2
}

func main() {
	s := fmt.Sprint(f())
	fmt.Print(s)
}
