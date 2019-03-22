//+build ignore

package main

func main() {
	var f func()
	{
		a := 1
		f = func() { print(a) }
	}
	f()
}
