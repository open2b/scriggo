// run

package main

import "fmt"

func f(s1, s2 string) string {
	return s1 + s2
}

func g(s string) int {
	return len(s)
}

func main() {
	b := "hey"
	fmt.Print(
		g(
			f("a", b),
		),
	)
}
