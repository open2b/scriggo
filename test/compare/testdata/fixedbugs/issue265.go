// run

package main

import "fmt"

func f() (int, int) {
	return 10, 20
}

func main() {
	fmt.Print(f())
}
