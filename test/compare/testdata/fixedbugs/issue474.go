// errorcheck

package main

func main() {
	switch v := interface{}(4).(type) { } // ERROR `v declared but not used`
}
