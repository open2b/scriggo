//+build ignore

package main

import "fmt"

func main() {
	{
		v := interface{}(int(0))
		_, ok := v.(int)
		if ok {
			fmt.Println("is int")
		} else {
			fmt.Println("is not int..")
		}
	}
	{
		v := interface{}(string("hello"))
		_, ok := v.(int)
		if ok {
			fmt.Println("is int")
		} else {
			fmt.Println("is not int..")
		}
	}
	{
		var i interface{} = "hello"

		s := i.(string)
		fmt.Println(s)

		s, ok := i.(string)
		fmt.Println(s, ok)

		f, ok := i.(float64)
		fmt.Println(f, ok)
	}
}
