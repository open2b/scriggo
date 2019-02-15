// Go version: go1.11.5

package template

import original "text/template"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"ExecError": reflect.TypeOf(original.ExecError{}),
	"FuncMap": reflect.TypeOf((original.FuncMap)(nil)),
	"HTMLEscape": original.HTMLEscape,
	"HTMLEscapeString": original.HTMLEscapeString,
	"HTMLEscaper": original.HTMLEscaper,
	"IsTrue": original.IsTrue,
	"JSEscape": original.JSEscape,
	"JSEscapeString": original.JSEscapeString,
	"JSEscaper": original.JSEscaper,
	"Must": original.Must,
	"New": original.New,
	"ParseFiles": original.ParseFiles,
	"ParseGlob": original.ParseGlob,
	"Template": reflect.TypeOf(original.Template{}),
	"URLQueryEscaper": original.URLQueryEscaper,
}
