// run

package main

import "fmt"

func f(s ...interface{}) {
	fmt.Printf("%T %T", s[0], s[1])
}

func g() (int, []int) {
	return 3, []int{1, 5}
}

func main() {
	f(g())
}
