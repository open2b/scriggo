// run

package main

import "fmt"

func main() {
	{
		_ = map[int][2]int{}[1]
	}
	{
		_ = map[int][2]int{
			10: [2]int{10, 20},
		}[10]
	}
	{
		m := map[int][2]int{}
		fmt.Println(m)
		fmt.Println(m[1])
	}
	{
		m := map[int][2]int{
			10: [2]int{10, 20},
		}
		fmt.Println(m)
		fmt.Println(m[10])
	}
	{
		_ = map[string][2]string{}["x"]
	}
	{
		_ = map[string][2][]int{
			"x": [2][]int{[]int{}, []int{4, 5, 6}},
		}["x"]
	}
	{
		m := map[string][2]string{}
		fmt.Println(m)
		fmt.Println(m["x"])
	}
	{
		m := map[string][2]string{
			"x": [2]string{"a", "b"},
		}
		fmt.Println(m)
		fmt.Println(m["x"])
	}
}
