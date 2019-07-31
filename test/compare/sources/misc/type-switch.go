// skip



package main

import (
	"fmt"
)

func main() {

	var v interface{} = 5

	switch v.(type) {
	case int:
		fmt.Println(v)
	}

	switch a := v.(type) {
	case int:
		fmt.Println(a, v)
	}

	switch a := 3; v.(type) {
	case int:
		fmt.Println(a, v)
	}

	switch a := 3; b := v.(type) {
	case int:
		fmt.Println(a, b, v)
	}

	switch v := interface{}(1); b := v.(type) {
	case int:
		fmt.Println(b, v)
	}

	switch a := interface{}(3); a := a.(type) {
	case int:
		fmt.Println(a)
	}

}
