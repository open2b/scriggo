// run

package main

func main() {
	var v uint
	_ = (1<<31)<<v + 'a'<<v + int64(1)
}
