// run

package main

import "fmt"

func f(args ...string) {
	fmt.Println(args[0])
}

func g() (string, string) {
	return "x", "y"
}

func main() {
	f(g())
}
