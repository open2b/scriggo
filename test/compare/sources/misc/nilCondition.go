//+build ignore

package main

import (
	"fmt"
)

func main() {

	var i interface{}
	if i != nil {
		panic("i != nil")
	}

	var i2 interface{} = []int(nil)
	i2 = []int(nil)
	_ = i2
	// if i2 == nil {
	// 	panic("i2 == nil")
	// }

	_, err := fmt.Println()
	if err != nil {
		panic("err != nil")
	}

	if nil != err {
		panic("nil != err")
	}

}
