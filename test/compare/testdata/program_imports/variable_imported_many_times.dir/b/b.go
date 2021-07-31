package b

import (
	"fmt"
	"variable_imported_many_times.dir/c"
)

func init() {
	fmt.Printf("b: c.C is %d\n", c.C)
}
