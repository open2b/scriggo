// run

package main

import "fmt"

func sum(nums ...int) {
	fmt.Print(nums, " ")
	nums[0] = -2
	fmt.Print(nums, " ")
}

func main() {
	sum(1, 2)
	sum(1, 2, 3)
}
