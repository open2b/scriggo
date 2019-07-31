

package main

var (
	A    = 1
	B, C = 2, 3
	D    int
	E    *int
)

func main() {
	var (
		A    = 1
		B, C = 2, 3
		D    int
		E    *int
	)
	_, _, _, _, _ = A, B, C, D, E
}
