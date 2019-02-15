// Go version: go1.11.5

package doc

import original "go/doc"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Example": reflect.TypeOf(original.Example{}),
	"Examples": original.Examples,
	"Filter": reflect.TypeOf((original.Filter)(nil)),
	"Func": reflect.TypeOf(original.Func{}),
	"IllegalPrefixes": &original.IllegalPrefixes,
	"IsPredeclared": original.IsPredeclared,
	"Mode": reflect.TypeOf(original.Mode(int(0))),
	"New": original.New,
	"Note": reflect.TypeOf(original.Note{}),
	"Package": reflect.TypeOf(original.Package{}),
	"Synopsis": original.Synopsis,
	"ToHTML": original.ToHTML,
	"ToText": original.ToText,
	"Type": reflect.TypeOf(original.Type{}),
	"Value": reflect.TypeOf(original.Value{}),
}
