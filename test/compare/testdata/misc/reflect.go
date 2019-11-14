// run

package main

import (
	"fmt"
	"reflect"
)

func main() {
	rv := reflect.ValueOf(10)
	interf := rv.Interface()
	i := interf.(int)
	fmt.Println(i)
}
