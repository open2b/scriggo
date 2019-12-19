// run

package main

import "fmt"

func main() {
	{
		for i := 0; i < 10; i++ {
			continue
			continue
			continue
		}
	}
	{
		for i := 0; i < 10; i++ {
			if i == -1 {
				continue
			}
			fmt.Println(i)
		}
	}
	{
		for i := 0; i < 10; i++ {
			if i%2 == 0 {
				continue
			}
			fmt.Println(i)
		}
	}
	{
		for i, v := range []string{"a", "b", "c"} {
			if v == "b" {
				continue
			}
			fmt.Println(i, v)
		}
	}
}
