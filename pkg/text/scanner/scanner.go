// Go version: go1.11.5

package scanner

import original "text/scanner"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Position": reflect.TypeOf(original.Position{}),
	"Scanner": reflect.TypeOf(original.Scanner{}),
	"TokenString": original.TokenString,
}
