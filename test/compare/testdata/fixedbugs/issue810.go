// run

package main

import "fmt"

type PkgT1 int

func main() {

	{
		type T1 int
		type T2 int
		_, ok := (interface{}(T1(0))).(T2)
		fmt.Println(ok)
	}

	{
		type T1 int
		type T2 string
		_, ok := (interface{}(T1(0))).(T2)
		fmt.Println(ok)
	}

	{
		type T1 int
		_, ok := (interface{}(T1(0))).(T1)
		fmt.Println(ok)
	}

	{
		_, ok := (interface{}(PkgT1(0))).(PkgT1)
		fmt.Println(ok)
	}

	{
		type T2 int
		_, ok := (interface{}(PkgT1(0))).(T2)
		fmt.Println(ok)
	}

	{
		_, ok := (interface{}(PkgT1(0))).(PkgT1)
		fmt.Println(ok)
	}

}
