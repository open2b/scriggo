// run

package main

import "fmt"

func main() {
	{
		a := []interface{}{
			1, 2, 3,
		}
		fmt.Println(a)
		a[1] = 5
		fmt.Println(a)
	}
	{
		s := []interface{}{10}
		s[0] = 20
	}
	{
		s := []interface{}{10}
		fmt.Printf("%v %T\n", s, s)
	}
	{
		s := []interface{}{10}
		s[0] = 20
	}
}
