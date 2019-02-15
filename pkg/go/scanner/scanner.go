// Go version: go1.11.5

package scanner

import original "go/scanner"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Error": reflect.TypeOf(original.Error{}),
	"ErrorHandler": reflect.TypeOf((original.ErrorHandler)(nil)),
	"ErrorList": reflect.TypeOf((original.ErrorList)(nil)),
	"Mode": reflect.TypeOf(original.Mode(uint(0))),
	"PrintError": original.PrintError,
	"Scanner": reflect.TypeOf(original.Scanner{}),
}
