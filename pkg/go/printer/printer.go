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
	"RawFormat": scrigo.Constant(original.RawFormat, nil),
	"SourcePos": scrigo.Constant(original.SourcePos, nil),
	"TabIndent": scrigo.Constant(original.TabIndent, nil),
	"UseSpaces": scrigo.Constant(original.UseSpaces, nil),
}
