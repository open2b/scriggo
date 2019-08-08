// run

package main

import "fmt"

func print(v interface{}) {
	fmt.Print(v, ",")
}

func main() {

	a := 27
	print(a) // 27
	a *= 6
	print(a) // 162
	a /= 3
	print(a) // 54
	a %= 16
	print(a) // 6
	a &= 10
	print(a) // 2
	a |= 21
	print(a) // 23
	a <<= 3
	print(a) // 184
	a >>= 4
	print(a) // 11

	print("-")

	a = 5
	print(a) // 5
	a += 3
	print(a) // 8
	a -= 2
	print(a) // 6
	a *= 6
	print(a) // 36
	a /= 3
	print(a) // 12
	a &= 7
	print(a) // 4
	a |= 3
	print(a) // 7
	a &^= 1
	print(a) // 6
	a <<= 2
	print(a) // 24
	a >>= 1
	print(a) // 12

}
