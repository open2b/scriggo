package main

import "fmt"

import "pkg1"

var MainA = MainB * 2
var MainB = 100

func main() {
	fmt.Println(MainA, MainB)
	fmt.Println(pkg1.Pkg1A, pkg1.Pkg1B)
}
