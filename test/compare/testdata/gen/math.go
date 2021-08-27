// skip

// This is not a test but a test generator.
// From this directory run the command 'go run math.go' to generate a test file.

package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
)

func main() {

	types := []string{"int8", "int16", "int32", "int64", "uint8", "uint16", "uint32", "uint64"}
	ops := []string{"+", "-", "*"}

	// Signed integers values.
	const min8, max8 = -1 << 7, 1<<7 - 1
	const min16, max16 = -1 << 15, 1<<15 - 1
	const min32, max32 = -1 << 16, 1<<16 - 1
	const min64, max64 = -1 << 63, 1<<63 - 1
	values8 := []int64{min8, min8 + 1, min8 + 2, -2, -1, 0, 1, 2, max8 - 2, max8 - 1, max8}
	values16 := []int64{min16, min16 + 1, min16 + 2, min8 - 1, min8, min8 + 1, -2, -1, 0, 1, 2,
		max8 - 1, max8, max8 + 1, max16 - 2, max16 - 1, max16}
	values32 := []int64{min32, min32 + 1, min32 + 2, min16 - 1, min16, min16 + 1, min8 - 1, min8, min8 + 1,
		-2, -1, 0, 1, 2, max8 - 1, max8, max8 + 1, max16 - 1, max16, max16 + 1, max32 - 2, max32 - 1, max32}
	values64 := []int64{min64, min64 + 1, min64 + 2, min32 - 1, min32, min32 + 1, min16 - 1, min16, min16 + 1,
		min8 - 1, min8, min8 + 1, -2, -1, 0, 1, 2, max8 - 1, max8, max8 + 1, max16 - 1, max16, max16 + 1,
		max32 - 1, max32, max32 + 1, max64 - 2, max64 - 1, max64}

	// Unsigned integers values.
	const maxU8 = 1<<8 - 1
	const maxU16 = 1<<16 - 1
	const maxU32 = 1<<32 - 1
	const maxU64 = 1<<64 - 1
	valuesU8 := []uint64{0, 1, 2, max8 - 2, max8 - 1, max8}
	valuesU16 := []uint64{0, 1, 2, max8 - 1, max8, max8 + 1, max16 - 2, max16 - 1, max16}
	valuesU32 := []uint64{0, 1, 2, max8 - 1, max8, max8 + 1, max16 - 1, max16, max16 + 1, max32 - 2, max32 - 1, max32}
	valuesU64 := []uint64{0, 1, 2, max8 - 1, max8, max8 + 1, max16 - 1, max16, max16 + 1, max32 - 1, max32, max32 - 1,
		max64 - 2, max64 - 1, max64}

	w := &bytes.Buffer{}

	fmt.Fprintf(w, "// run\n\n")
	fmt.Fprintf(w, "// This file has been auto-generated.\n\n")
	fmt.Fprintf(w, "package main\n\nfunc main(){\n")
	for _, typ := range types {

		isUnsigned := strings.HasPrefix(typ, "u")

		initial := "-128"
		if isUnsigned {
			initial = "0"
		}

		fmt.Fprintf(w, "\n\t// %s\n", typ)
		fmt.Fprintf(w, "\t"+`for i := %s(%s); ; i++ {`+"\n", typ, initial)
		for _, op := range ops {
			if isUnsigned {
				var consts []uint64
				switch typ {
				case "uint8":
					consts = valuesU8
				case "uint16":
					consts = valuesU16
				case "uint32":
					consts = valuesU32
				case "uint64":
					consts = valuesU64
				}
				for _, c := range consts {
					fmt.Fprintf(w, "\t\tprintln(\"i %s (%v) = \", i %s (%v))\n", op, c, op, c)
					fmt.Fprintf(w, "\t\tprintln(\"(%v) %s i = \", (%v) %s i)\n", c, op, c, op)
				}
			} else {
				var consts []int64
				switch typ {
				case "int8":
					consts = values8
				case "int16":
					consts = values16
				case "int32":
					consts = values32
				case "int64":
					consts = values64
				}
				for _, c := range consts {
					fmt.Fprintf(w, "\t\tprintln(\"i %s (%v) = \", i %s (%v))\n", op, c, op, c)
					fmt.Fprintf(w, "\t\tprintln(\"(%v) %s i = \", (%v) %s i)\n", c, op, c, op)
				}
			}
		}

		fmt.Fprintf(w, "\t\tif i == 127 {\n\t\t\tbreak\n\t\t}\n\t}\n")
	}
	fmt.Fprintf(w, "}\n")

	err := os.WriteFile("../misc/math-generated.go", w.Bytes(), 0755)
	if err != nil {
		panic(err)
	}

}
