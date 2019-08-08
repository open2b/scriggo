// run

package main

import "fmt"

func main() {
	fmt.Print(interface{}(1).(int))
	i := interface{}(2).(int)
	fmt.Print(i)
	fmt.Print(interface{}(5).(int) * interface{}(7).(int))
}
