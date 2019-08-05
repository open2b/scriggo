// run

package main

import "fmt"

func main() {

	// Condition is a boolean constant.
	if true {
		fmt.Println("true")
	} else {
		panic("error")
	}

	if false {
		panic("error")
	} else {
		fmt.Println("false")
	}

	if 43 == 43 {
		fmt.Println("43 == 43")
	} else {
		panic("error")
	}

	if !true == (56 > 100) {
		fmt.Println("!true == 56 > 100")
	} else {
		panic("error")
	}

	// Binary operations where one of the operands is nil.
	vNil := []int(nil)
	if vNil == nil {
		fmt.Println("vNil == nil")
	}
	if vNil != nil {
		panic("vNil != nil")
	}
	if nil == vNil {
		fmt.Println("nil == vNil")
	}
	if nil != vNil {
		panic("nil != vNil")
	}
	vNotNil := []int{}
	if vNotNil == nil {
		panic("vNotNil == nil")
	}
	if vNotNil != nil {
		fmt.Println("vNotNil != nil")
	}
	if nil == vNotNil {
		panic("nil == vNotNil")
	}
	if nil != vNotNil {
		fmt.Println("nil != vNotNil")
	}
}
