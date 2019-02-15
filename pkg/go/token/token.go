// Go version: go1.11.5

package token

import original "go/token"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"File": reflect.TypeOf(original.File{}),
	"FileSet": reflect.TypeOf(original.FileSet{}),
	"Lookup": original.Lookup,
	"NewFileSet": original.NewFileSet,
	"Pos": reflect.TypeOf(original.Pos(int(0))),
	"Position": reflect.TypeOf(original.Position{}),
	"Token": reflect.TypeOf(original.Token(int(0))),
}
