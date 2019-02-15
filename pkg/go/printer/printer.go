// Go version: go1.11.5

package printer

import original "go/printer"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"CommentedNode": reflect.TypeOf(original.CommentedNode{}),
	"Config": reflect.TypeOf(original.Config{}),
	"Fprint": original.Fprint,
	"Mode": reflect.TypeOf(original.Mode(uint(0))),
}
