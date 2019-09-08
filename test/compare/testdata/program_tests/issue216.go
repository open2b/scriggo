// run

package main

import "fmt"

func main() {
	for i := 0; i < 26; i++ {
		c := string([]byte{byte(i) + 'a'})
		fmt.Print(c)
	}
}
