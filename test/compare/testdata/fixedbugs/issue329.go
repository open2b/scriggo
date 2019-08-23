// run

package main

import "fmt"

func typeof(i interface{}) string {
	return fmt.Sprintf("%T", i)
}

func main() {
	{
		var v1 (int)
		var v2 (string)
		var v3 ([]int)
		var v4 (map[string]([](int)))
		fmt.Println(typeof(v1))
		fmt.Println(typeof(v2))
		fmt.Println(typeof(v3))
		fmt.Println(typeof(v4))
	}
	{
		var v1 (int) = 2
		var v2 (string) = ("hello")
		var v3 ([]int) = [](int){1, 2, 3}
		var v4 (map[string][]int) = map[(string)][]int{}
		fmt.Println(typeof(v1))
		fmt.Println(typeof(v2))
		fmt.Println(typeof(v3))
		fmt.Println(typeof(v4))
	}
	{
		const v1 (int) = 2
		const v2 (string) = ("hello")
		fmt.Println(typeof(v1))
		fmt.Println(typeof(v2))
	}
	{
		_ = func() {}
		_ = func(int) {}
		_ = func(string) int { return 0 }
		_ = func(string) int { return 0 }
		_ = func(m map[string](int), a string) {}
	}
}
