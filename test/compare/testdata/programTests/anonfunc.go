// run

package main

import "fmt"

func f(int) {}

func g(string, string) int {
	return 10
}

func main() {
	f(3)
	v := g("a", "b")
	fmt.Println(v)
}
