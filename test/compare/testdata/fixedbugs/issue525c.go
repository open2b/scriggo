// run

package main

import "fmt"

func main() {
	slice := []byte("000000")
	{
		copy(slice, "12346")
		fmt.Println(slice)
	}
	slice = []byte("000000")
	{
		s := "hello"
		copy(slice, s)
		fmt.Println(slice)
	}
	slice = []byte("000000")
	{
		type String string
		s := String("hello")
		copy(slice, s)
		fmt.Println(slice)
	}
	slice = []byte("000000")
	{
		type String string
		s := String("hello")
		copy([]byte{}, s)
		fmt.Println(slice)
	}
}
