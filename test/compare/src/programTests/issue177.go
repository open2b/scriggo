// run

package main

import "fmt"

func main() {
	_, err := fmt.Println()
	if err != nil {
		panic(err)
	}
}
