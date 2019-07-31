

package main

import "fmt"

// The inizialization order is: [B, A], because A depends on B.

var A = B
var B = 10

func main() {
	fmt.Println(A)
	fmt.Println(B)
}
