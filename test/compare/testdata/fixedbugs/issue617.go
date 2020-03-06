// run

package main

import "fmt"

func main() {
	fmt.Println("---------")
	{
		_ = append([]byte{}, "1"...)
	}
	fmt.Println("---------")
	{
		s := "1"
		_ = append([]byte{}, s...)
	}
	fmt.Println("---------")
	{
		fmt.Println(append([]byte{}, ""...))
		fmt.Println(append([]byte{}, "1"...))
		fmt.Println(append([]byte{97, 98, 99}, "1"...))
		fmt.Println(append([]byte{}, "hello"...))
	}
	fmt.Println("---------")
	{
		v1 := append([]byte{}, ""...)
		fmt.Println(v1)
		v2 := append([]byte{}, "1"...)
		fmt.Println(v2)
		v3 := append([]byte{97, 98, 99}, "1"...)
		fmt.Println(v3)
		v4 := append([]byte{}, "hello"...)
		fmt.Println(v4)
	}
	fmt.Println("---------")
	{
		const (
			c1 = ""
			c2 = " "
			c3 = "1"
			c4 = "hello"
		)
		fmt.Println(append([]byte{}, c1...))
		fmt.Println(append([]byte{}, c2...))
		fmt.Println(append([]byte{97, 98, 99}, c3...))
		fmt.Println(append([]byte{}, c4...))
	}
	fmt.Println("---------")
	{
		var (
			v1 = ""
			v2 = " "
			v3 = "1"
			v4 = "hello"
		)
		fmt.Println(append([]byte{}, v1...))
		fmt.Println(append([]byte{}, v2...))
		fmt.Println(append([]byte{97, 98, 99}, v3...))
		fmt.Println(append([]byte{}, v4...))
	}
	fmt.Println("---------")
	{
		type SliceByte []byte
		fmt.Println([]byte(append(SliceByte{}, ""...)))
		fmt.Println([]byte(append(SliceByte{}, "10"...)))
		fmt.Println([]byte(append([]byte{97, 98, 99}, "x"...)))
		fmt.Println([]byte(append(SliceByte{}, "hola"...)))
	}
}
