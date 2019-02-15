// Go version: go1.11.5

package template

import original "html/template"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"CSS": reflect.TypeOf(""),
	"Error": reflect.TypeOf(original.Error{}),
	"ErrorCode": reflect.TypeOf(original.ErrorCode(int(0))),
	"FuncMap": reflect.TypeOf((original.FuncMap)(nil)),
	"HTML": reflect.TypeOf(""),
	"HTMLAttr": reflect.TypeOf(""),
	"HTMLEscape": original.HTMLEscape,
	"HTMLEscapeString": original.HTMLEscapeString,
	"HTMLEscaper": original.HTMLEscaper,
	"IsTrue": original.IsTrue,
	"JS": reflect.TypeOf(""),
	"JSEscape": original.JSEscape,
	"JSEscapeString": original.JSEscapeString,
	"JSEscaper": original.JSEscaper,
	"JSStr": reflect.TypeOf(""),
	"Must": original.Must,
	"New": original.New,
	"ParseFiles": original.ParseFiles,
	"ParseGlob": original.ParseGlob,
	"Srcset": reflect.TypeOf(""),
	"Template": reflect.TypeOf(original.Template{}),
	"URL": reflect.TypeOf(""),
	"URLQueryEscaper": original.URLQueryEscaper,
}
