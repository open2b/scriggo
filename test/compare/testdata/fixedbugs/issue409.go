// run

package main

import (
	"reflect"
)

func main() {
	rv := reflect.ValueOf("str")
	rv.IsValid()
}