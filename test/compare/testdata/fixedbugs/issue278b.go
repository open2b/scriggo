// run

package main

func f() (x int) {
	defer func() func() {
		return func() {
			println(x)
		}
	}()()
	return 42
}

func main() {
	f()
}
