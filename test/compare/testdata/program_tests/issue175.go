// run

package main

func main() {
	switch u := interface{}(2).(type) {
	case int:
		_ = u * 2
	}
}
