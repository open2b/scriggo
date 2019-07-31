// runcompare

package main

import "fmt"

// Constant B depends on A, so constant initialization order is: [B, A].

const A = B
const B = 20

func main() {
	fmt.Println(A)
	fmt.Println(B)
}
