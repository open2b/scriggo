package a

import (
	"fmt"
	"variable_imported_many_times.dir/c"
)

func init() {
	fmt.Printf("a: c.C is %d (before increment)\n", c.C)
	c.C = c.C + 10
	fmt.Printf("a: c.C is %d (after increment)\n", c.C)
}
