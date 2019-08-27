//+build !windows

// paniccheck -time=10ms

package main

import "time"

func main() {
	time.Sleep(3 * time.Millisecond)
}
