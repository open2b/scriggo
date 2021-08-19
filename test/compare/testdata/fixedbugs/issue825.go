// errorcheck

package main

func main() {
	if true { } else ; // ERROR `else must be followed by if or statement block`
}