// run

package main

import "fmt"

func main() {
	{
		iface := interface{}(int(1))
		switch v := iface.(type) {
		default:
			_ = v
			fmt.Print("default")
		case int8:
			_ = v
			fmt.Print("int8")
		}
	}
	{
		iface := interface{}(true)
		switch v := iface.(type) {
		default:
			_ = v
			fmt.Print("default")
		case bool:
			_ = v
			fmt.Print("bool")
		case int64:
			_ = v
			fmt.Print("int64")
		}
	}
}
