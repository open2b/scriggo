// errorcheck

package main

import "fmt"   // ERROR "^imported and not used: \"fmt\"$"
import p "os"  // ERROR "^imported and not used: \"os\" as p$"
import . "net" // ERROR "^imported and not used: \"net\"$"
import _ "strings"

func main() {}
