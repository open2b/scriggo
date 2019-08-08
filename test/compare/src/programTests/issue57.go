// run

package main

import "fmt"

func main() {
	v := interface{}(3)
	switch u := v.(type) {
	default:
		fmt.Println(u)
	}
}
