// skip : see https://github.com/open2b/scriggo/issues/369

// run

package main

import "fmt"

var a interface{}

const a1 = len("")
const a2 = len("abc")
const a3 = len([0]int{})
const a4 = len((*[0]int)(nil))
const a5 = len([5]int{})
const a6 = len((*[5]int)(nil))
const a7 = len([5]int{1})
const a8 = len([3]int{1, 2, 3})
const a9 = len([][7]int{}[0])
const a10 = len(([][2]int)(nil)[0])
const a11 = len(map[string][2]int{}["a"])
const a12 = len(make([][1]int, 1)[0])
const a13 = len([1]int{len("a")})
const a14 = len([1]int{cap([2]int{})})
const a15 = len([1]int{a.(int)})
const a16 = len([3]int{real(1i), imag(1i), complex(1, 0)})
const a17 = len(struct{ a [2]int }{}.a)

func main() {

	assert(1, a1, 0)
	assert(2, a2, 3)
	assert(3, a3, 0)
	assert(4, a4, 0)
	assert(5, a5, 5)
	assert(6, a6, 5)
	assert(7, a7, 5)
	assert(8, a8, 3)
	assert(9, a9, 7)
	assert(10, a10, 2)
	assert(11, a11, 2)
	assert(12, a12, 1)
	assert(13, a13, 1)
	assert(14, a14, 1)
	assert(15, a15, 1)
	assert(16, a16, 3)
	assert(17, a17, 2)

	a = 8

	var b3 = [0]int{}
	var b4 = (*[0]int)(nil)
	var b5 = [5]int{}
	var b6 = (*[5]int)(nil)
	var b7 = [5]int{1}
	var b8 = [3]int{1, 2, 3}
	var b9 = [][7]int{{}}[0]
	var b11 = map[string][2]int{}["a"]
	var b12 = make([][1]int, 1)[0]
	var b13 = [1]int{len("a")}
	var b14 = [1]int{cap([2]int{})}
	var b15 = [1]int{a.(int)}
	var b16 = [3]int{real(1i), imag(1i), complex(1, 0)}
	var b17 = struct{ a [2]int }{}.a

	const _ = len(b3)
	const _ = len(b4)
	const _ = len(b5)
	const _ = len(b6)
	const _ = len(b7)
	const _ = len(b8)
	const _ = len(b9)
	const _ = len(b11)
	const _ = len(b12)
	const _ = len(b13)
	const _ = len(b14)
	const _ = len(b15)
	const _ = len(b16)
	const _ = len(b17)

}

func assert(n, a, b int) {
	if a != b {
		panic(fmt.Errorf("test %d: expected %d, got %d", n, b, a))
	}
}
