// run

package main

type T struct{}

func main() {
	delete(map[T]int{}, T{})
}
