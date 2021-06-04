// run

package main

type S struct{}

type T struct{ S }

var a3 = T{S{}}

func main() {}
