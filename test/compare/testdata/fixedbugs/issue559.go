// errorcheck

package main

func main() {
    if nil { } // ERROR `use of untyped nil`
}