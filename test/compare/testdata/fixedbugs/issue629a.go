// skip

package main

import "fmt"

func main() {
	addresses := [3]*int{}
	for i, v := range [3]int{10, 20, 30} {
		addresses[i] = &v
	}
	fmt.Println(addresses[0] == addresses[1])
	fmt.Println(addresses[0] == addresses[2])
}
