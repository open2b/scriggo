// paniccheck

package main

func main() {
	var x uint8 = 4
	var y = [1]uint32{1}
	print(y[128+x])
}