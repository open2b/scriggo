// run

package main

import "fmt"

func F(_ interface{}) (int, int) {
	v := 300
	return v, 200
}

func main() {
	fmt.Println(F(100))
}
