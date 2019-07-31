// runcompare

package main

import "fmt"

func f() {
	{
	}
	fmt.Println("f1")
	{
		fmt.Println("f2")
	}
}

func main() {
	a := 1
	{
		fmt.Println(a)
		a := 2
		fmt.Println(a)
		{
			fmt.Println(a)
			a := 3
			fmt.Println(a)
		}
		fmt.Println(a)
	}
	fmt.Println(a)

	if true {
		if 0 == 0 {
			{
				{
					{
					}
					{
						fmt.Println("inner")
					}
				}
				{
				}
			}
		}
		f()
	}
}
