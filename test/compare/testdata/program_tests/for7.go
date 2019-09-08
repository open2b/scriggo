// run

package main

import "fmt"

func main() {

	for i, c := range "Hello" {
		fmt.Print(i, c, ",")
	}

}
