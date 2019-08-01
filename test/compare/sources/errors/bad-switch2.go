// errorcheck

package main

func main() {
	switch interface{}(0).(type) {
		case 1: // ERROR `1 (type untyped number) is not a type`
	}
}
