// run

package main

func main() {
	c := interface{}(true) == 8
	_ = c

	switch interface{}(true) {
	case true:
	}
}
