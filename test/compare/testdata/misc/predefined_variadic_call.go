// run

package main

import "fmt"

func main() {
	s := []interface{}{1, 2, 3, 4, 5}
	str := fmt.Sprintf("hello %d, %d, %d, %d, %d", s...)
	fmt.Println(str)
}
