// run

package main

import "fmt"

func main() {
	defer func() {
		fmt.Println("end")
	}()
	fmt.Println("start")
}
