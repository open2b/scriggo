// run

package main

import "fmt"

func F(_ interface{}) (int, int) {
	return 300, 200
}

func main() {
	fmt.Println(F(100))
}
