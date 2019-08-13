// run

package main

import "fmt"

func main() {
	if !false {
		if !!false {
		} else if !false {
			fmt.Println("here")
		}
	}
	a := 3
	if true {
		a = 4
	} else if true {
		fmt.Println("shouldn't print this...")
	}
	if a == 4 {
		fmt.Println("a is 4")
	}
	{
		var v interface{} = 1
		if v == nil {
			fmt.Println("v == nil")
		} else {
			fmt.Println("v != nil")
		}
		if v != nil {
			fmt.Println("v != nil")
		} else {
			fmt.Println("v == nil")
		}
		if nil != v {
			fmt.Println("nil != v")
		}
	}
	{
		v1 := []int{1, 2, 3}
		if v1 == nil {
			panic("should not be nil")
		} else {
			fmt.Println(v1)
		}
		v2 := []int(nil)
		if v2 != nil {
			panic("should be nil")
		} else {
			fmt.Println(v2)
		}
	}

}
