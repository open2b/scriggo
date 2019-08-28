// skip

// paniccheck

package main

func main() {
	defer func() {
		recover()
		panic(2)
	}()
	panic(1)
}
