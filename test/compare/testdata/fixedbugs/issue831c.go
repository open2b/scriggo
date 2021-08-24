// run

package main

import "fmt"

type Slice []int

func main() {
	var c Slice = Slice{1, 2, 3, 4, 5}
	var i interface{} = c[0:2]
	_, ok := i.(Slice)
	fmt.Println(ok)
}
