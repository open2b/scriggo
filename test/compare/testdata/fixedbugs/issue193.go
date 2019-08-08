// run

package main

import "fmt"

func main() {
	switch (*int)(nil) {
	case nil:
		fmt.Print("nil!")
	}
	switch []int(nil) {
	case nil:
		fmt.Print("nil!")
	}
}
