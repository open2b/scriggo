// run

package main

import "fmt"

func main() {
	c1 := []int(nil) != nil
	fmt.Println(c1)

	c2 := []int(nil) == nil
	fmt.Println(c2)

	c3 := nil == []int(nil)
	fmt.Println(c3)

	c4 := interface{}(nil) == nil
	fmt.Println(c4)

	c5 := interface{}(nil) != nil
	fmt.Println(c5)

	// Some inline conditions.
	fmt.Println(nil == interface{}(nil))
	fmt.Println(nil != interface{}(nil))
	fmt.Println(interface{}(nil) == interface{}(nil))
	fmt.Println([]int(nil) == nil || []int(nil) != nil)
	fmt.Println(interface{}([]int(nil)) == interface{}(nil))

}
