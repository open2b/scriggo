// run

package main

import "fmt"

func f1() (a int, b int) {
	a = 10
	b = 20
	return
}

func f2() (a int, b int) {
	return 30, 40
}

func f3(bool) (a int, b int, c string) {
	a = 70
	b = 80
	c = "str2"
	return
}

func f4() (a int, b int, c float64) {
	a = 90
	c = 100
	return
}

func main() {
	v11, v12 := f1()
	v21, v22 := f2()
	v31, v32, v33 := f3(true)
	v41, v42, v43 := f3(false)
	v51, v52, v53 := f4()
	fmt.Println(v11, v12)
	fmt.Println(v21, v22)
	fmt.Println(v31, v32, v33)
	fmt.Println(v41, v42, v43)
	fmt.Println(v51, v52, v53)
}
