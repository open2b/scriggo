// errorcheck

package main

// TODO: the returned error message is a bit different from the one returned by
// gc.

type int + // ERROR `syntax error: unexpected +, expecting type`

func main() {
	
}