// skip

// This is not a test but a test generator.
// From this directory run the command 'go run math.go' to generate a test file.

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
)

func main() {

	types := []string{"int8", "int16", "int32", "int64", "uint8", "uint16", "uint32", "uint64"}
	consts := []int{-127, -32, -1, 0, 1, 32, 126}
	ops := []string{"+", "-", "*"}

	w := &bytes.Buffer{}

	fmt.Fprintf(w, "// run\n\n")
	fmt.Fprintf(w, "// This file has been auto-generated.\n\n")
	fmt.Fprintf(w, "package main\n\nfunc main(){\n")
	for _, typ := range types {

		isUnsigned := strings.HasPrefix(typ, "u")

		initial := "-127"
		if isUnsigned {
			initial = "0"
		}

		fmt.Fprintf(w, "\n\t// %s\n", typ)
		fmt.Fprintf(w, "\t"+`for i := %s(%s); i <= 126; i++ {`+"\n", typ, initial)
		for _, op := range ops {
			for _, c := range consts {
				if isUnsigned && c < 0 {
					continue
				}
				fmt.Fprintf(w, "\t\tprintln(\"i %s (%v) = \", i %s (%v))\n", op, c, op, c)
				fmt.Fprintf(w, "\t\tprintln(\"(%v) %s i = \", (%v) %s i)\n", c, op, c, op)
			}
		}
		fmt.Fprintf(w, "\t}\n")
	}
	fmt.Fprintf(w, "}\n")

	err := ioutil.WriteFile("../misc/math-generated.go", w.Bytes(), 0755)
	if err != nil {
		panic(err)
	}

}
