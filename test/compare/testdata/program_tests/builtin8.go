// run

package main

import (
	"fmt"
)

func main() {
	m := map[string]string{"key": "value", "key2": "value2"}
	fmt.Print(m, ",")
	delete(m, "key")
	fmt.Print(m)
}
