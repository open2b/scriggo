// run

package main

var x = "package-level"

func main() {
	{
		var x string
		x = "scope"
		_ = x
	}
	println(x)
}
