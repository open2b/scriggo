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

	if len("432") == 3 {
		fmt.Print(`len("432") == 3`)
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

	// Binary operations where one of the operands is the len builtin call with
	// a string argument (only len builtin calls with a string argument are
	// optimized in vm).
	if l := 3; len("str") == l {
		fmt.Println(`l := 3; len("str") == l`)
	} else {
		panic("error")
	}
	if l := 3; len("str") != l {
		panic("error")
	} else {
		fmt.Println(`l := 3; len("str") != l is false`)
	}

	// Others.
	if l := 10; len([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}) > l {
		fmt.Println(`greater`)
	} else {
		panic("error")
	}
}
