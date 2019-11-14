// run

package main

import "errors"

func main() {
	var x error
	px := &x
	*px = errors.New("message")
}
