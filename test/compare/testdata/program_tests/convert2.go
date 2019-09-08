// run

package main

import (
	"fmt"
)

func main() {
	i64 := int64(100)
	i := int(i64)
	interf := interface{}(i64)
	fmt.Print(i64, i, interf)
	fmt.Printf("%T %T %T", i64, i, interf)
}
