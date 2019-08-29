// run

package main

import (
	"errors"
	"fmt"
	"reflect"
)

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
	{
		_ = interface{}(errors.New("test")).(error)
	}
	{
		rv := reflect.ValueOf(fmt.Print)
		interf := rv.Interface()
		p := interf.(func(...interface{}) (int, error))
		p(1, 2, "x")
	}
}
