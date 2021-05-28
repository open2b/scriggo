// run

package main

import "fmt"

func g(e interface{}) (int, int) {
	return 10, 20
}

func f(d interface{}) (int, int) {
	return g(d)
	return 30, 40
}

func main() {
	fmt.Println(f(50))
}
