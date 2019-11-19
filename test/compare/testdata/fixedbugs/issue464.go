// run

package main

func main() {
	{
		_ = (func())(nil)
	}
	{
		f := (func(int, string) []int)(nil)
		_ = f
	}
}
