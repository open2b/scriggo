//+build ignore

package main

import "fmt"

func main() {
	for i := range []int{} {
		_ = i
	}
	for k, v := range map[string](map[string](map[int]int)){} {
		_ = v
		_ = k
	}
	for {
		break
	}
	for i := -3; i < 0; i++ {
		fmt.Println(i)
	}
	sum := 0
	for x := 10; true; {
		sum += x
		if sum > 50 {
			break
		}
	}
	fmt.Println(sum)
	{
		items := make([]map[int]int, 10)
		for i := range items {
			items[i] = make(map[int]int, 1)
			items[i][1] = 2
		}
		fmt.Println(items)
	}
}
