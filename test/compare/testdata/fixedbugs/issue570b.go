// run

package main

import "fmt"

func main() {
	slice := []int{42, 43, 44}
	for i, elem := range slice {
		func() {
			fmt.Println(i, elem)
			_ = &i
			_ = &elem
		}()
	}
}
