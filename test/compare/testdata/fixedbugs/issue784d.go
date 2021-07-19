// run

package main

import "fmt"

func f(args ...int) {
	fmt.Println(args[0])
}

func g() (int, int) {
	return 1, 2
}

func main() {
	f(g())
}
