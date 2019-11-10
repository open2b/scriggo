// errorcheck

package main

import "io/ioutil"

func main() {
	ioutil.ReadFile("test.txt", nil) // ERROR `too many arguments in call to ioutil.ReadFile`
	_ = ioutil.Discard
}
