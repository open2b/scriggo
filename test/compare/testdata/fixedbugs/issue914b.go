// run

package main

func main() {
	x := []string{"one", "two"}
	y := x
	x = nil
	println("X is nil:", x == nil)
	println("Y is nil:", y == nil)
}
