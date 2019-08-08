// run

package main

import "fmt"

func pair() (int, string) {
	return 10, "zzz"
}

func main() {
	var a, b = 1, 2
	var c, d int
	var e, f = pair()
	fmt.Print(a, b, c, d, e, f)
}
