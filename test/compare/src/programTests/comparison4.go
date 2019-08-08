// run

package main

import "fmt"

func main() {
	_, err := fmt.Println()
	if err != nil {
		fmt.Print("error")
	} else {
		fmt.Print("no errors")
	}
}
