// run

package main

import "fmt"

func main() {
	m := make(map[interface{}]int)
	key := []int{1, 2, 3}
	defer func() {
		r := recover()
		fmt.Println("recovered from:", r)
	}()
	delete(m, key)
}
