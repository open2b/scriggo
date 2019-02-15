// Go version: go1.11.5

package parser

import original "go/parser"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Mode": reflect.TypeOf(original.Mode(uint(0))),
	"ParseDir": original.ParseDir,
	"ParseExpr": original.ParseExpr,
	"ParseExprFrom": original.ParseExprFrom,
	"ParseFile": original.ParseFile,
}
