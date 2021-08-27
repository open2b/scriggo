// errorcheck

package main

import (
	"os"
)

func main() {
	os.ReadFile("test.txt", nil) // ERROR `too many arguments in call to os.ReadFile`
	_ = os.ReadFile
}
