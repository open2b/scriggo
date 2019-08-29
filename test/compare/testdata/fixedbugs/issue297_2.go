// errorcheck

package main

func main() {

	var b = []int{1}
	_ = b

	_ = []int{ 5; } // ERROR `unexpected semicolon, expecting comma or }`
	_ = []int{ b[0]; } // ERROR `unexpected semicolon, expecting comma or }`
	_ = []int{ f(); } // ERROR `unexpected semicolon, expecting comma or }`

}

func f() int {
	return 0
}
