package main

import aNewName "named_imports.dir/a"
import . "named_imports.dir/b"

func main() {
	aNewName.A()
	B()
}
