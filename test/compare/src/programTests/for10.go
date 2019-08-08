// run

package main

import "fmt"

func main() {
	sum := 0
	i := 0
	for i = 0; i < 10; i++ {
		fmt.Print("i=", i, ",")
		sum = sum + 2
	}
	fmt.Print("sum=", sum)
}
