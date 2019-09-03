// skip

// paniccheck

package main

func main() {
	defer func() {
		defer func() {
			recover()
		}()
		defer panic(2)
	}()
	defer panic(1)
}
