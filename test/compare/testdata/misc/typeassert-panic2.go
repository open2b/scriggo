// paniccheck

package main

import "fmt"

func main() {
	var i interface{} = 10
	fmt.Println(i.(fmt.Formatter))
}
