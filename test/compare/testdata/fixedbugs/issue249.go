// errorcheck

package main

func main() {
	const a uint64 = 5

	var b int64 = a ; _ = b // ERROR `cannot use a (type uint64) as type int64 in assignment`
}
