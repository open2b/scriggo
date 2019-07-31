// runcompare

package main

import (
	"fmt"
)

const (
	Nanosecond  float64 = 1
	Microsecond         = 1000 * Nanosecond
	Millisecond         = 1000 * Microsecond
	Second              = 1000 * Millisecond
	Minute              = 60 * Second
	Hour                = 60 * Minute
)

const (
	Sunday int = iota
	Monday
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
	numberOfWeekdays
)

func main() {
	fmt.Println(Nanosecond, Millisecond, Minute, Hour)
	fmt.Println(Sunday, Thursday)
	fmt.Println("number of weekdays:", numberOfWeekdays)
}
