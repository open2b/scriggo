// run

package main

func p() bool {
	panic("error")
}

func main() {
	T := true
	F := false

	v := false

	v = F && p()
	v = T || p()
	v = (F || F) && p()
	v = (T && T) || p()
	v = F && p() && p()
	v = F || T || p()
	v = T && F && p()

	_ = v
}
