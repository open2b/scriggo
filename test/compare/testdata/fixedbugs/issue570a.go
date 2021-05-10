// run

package main

func main() {
	for i := range []int{0} {
		func() { print(i) }()
	}
}
