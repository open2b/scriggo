// compile

package main

import "fmt"

type Int int

func Sum(a, b Int) Int {
	return a + b
}

func main() {
	var a, b Int = 30, 12
	fmt.Println(Sum(a, b))
}
