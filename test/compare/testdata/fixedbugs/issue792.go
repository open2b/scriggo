// run

package main

import "fmt"

func g() (int, string, string) {
	return 5, "xx", "yy"
}

func f(a int, b ...string) (int, []string) {
	return a, b
}

func main() {
	x, y := f(g())
	fmt.Println(x)
	fmt.Println(y)
}
