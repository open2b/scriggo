// Go version: go1.11.5

package syntax

import original "regexp/syntax"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Compile": original.Compile,
	"EmptyOp": reflect.TypeOf(original.EmptyOp(uint8(0))),
	"EmptyOpContext": original.EmptyOpContext,
	"Error": reflect.TypeOf(original.Error{}),
	"ErrorCode": reflect.TypeOf(""),
	"Flags": reflect.TypeOf(original.Flags(uint16(0))),
	"Inst": reflect.TypeOf(original.Inst{}),
	"InstOp": reflect.TypeOf(original.InstOp(uint8(0))),
	"IsWordChar": original.IsWordChar,
	"Op": reflect.TypeOf(original.Op(uint8(0))),
	"Parse": original.Parse,
	"Prog": reflect.TypeOf(original.Prog{}),
	"Regexp": reflect.TypeOf(original.Regexp{}),
}
