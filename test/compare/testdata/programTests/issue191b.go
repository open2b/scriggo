// run

package main

import "fmt"

func main() {
	switch u := interface{}(2).(type) {
	case int:
		fmt.Print(u * 2)
	}
}
