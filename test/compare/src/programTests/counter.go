// run

package main

import (
	"fmt"
)

func main() {
	count := 0
loop:
	fmt.Print(",")
	if count > 9 {
		goto end
	}
	fmt.Print(count)
	count++
	goto loop
end:
}
