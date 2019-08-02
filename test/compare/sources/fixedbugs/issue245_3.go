// run

package main

import "fmt"

func main() {
	m1 := map[int]int{0: 32}
	m1[0]++
	fmt.Println(m1)

	m2 := make(map[int]int)
	m2[0] = -2
	m2[0]++
	fmt.Println(m2)

	m3 := make(map[string]float64)
	m3["x"] = 3.0 / 2.0
	m3["x"]--
	fmt.Println(m3)
}
