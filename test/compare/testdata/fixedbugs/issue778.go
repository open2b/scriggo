// errorcheck

package main

const a // ERROR "missing value in const declaration"

const a, b = 5 // ERROR "missing value in const declaration"

const ( a; b = 5 ) // ERROR "missing value in const declaration"

const ( a = 5; b, c ) // ERROR "missing value in const declaration"

const ( a = 5; b; c )

const a = 2, 3 // ERROR "extra expression in const declaration"

const ( a = 2; b = 3, 4 ) // ERROR "extra expression in const declaration"

func main() { }
