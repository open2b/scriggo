// runcompare

package main

import "fmt"

func main() {

	for i := 0; i < 2; i++ {
		i := "hi"
		fmt.Println(i)
	}

	var i string = "empty"
	for i := range []int{4, 5, 6} {
		fmt.Println(i)
		i := "hi"
		fmt.Println(i)
	}
	fmt.Println(i)

	count := 3
	for count > 0 {
		count--
		fmt.Println(count)
	}

}
