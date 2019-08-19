// run

package main

import (
	"fmt"
)

func main() {
	{
		a := [3]int{1, 2, 3}
		pa := &a
		fmt.Println(pa[1])
	}
	{
		var a [2]float64
		pa := &a
		fmt.Println(pa[0])
		fmt.Println((*pa)[0])
	}
}
