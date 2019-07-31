// runcmp

package main

import "fmt"

func main() {
	a := "external"

	if a := 3; a > 10 {
	} else if a := "elseif"; len(a) > 2 {
		a := []int{1, 2, 3}
		fmt.Println(a)
	} else {
		a := map[string]int{}
		_ = a
	}
	fmt.Println(a)

}
