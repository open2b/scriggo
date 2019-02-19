// Go version: go1.11.5

package template

import "reflect"
import original "html/template"
import "scrigo"

var Package = scrigo.Package{
	"CSS": reflect.TypeOf(""),
	"ErrAmbigContext": scrigo.Constant(original.ErrAmbigContext, nil),
	"ErrBadHTML": scrigo.Constant(original.ErrBadHTML, nil),
	"ErrBranchEnd": scrigo.Constant(original.ErrBranchEnd, nil),
	"ErrEndContext": scrigo.Constant(original.ErrEndContext, nil),
	"ErrNoSuchTemplate": scrigo.Constant(original.ErrNoSuchTemplate, nil),
	"ErrOutputContext": scrigo.Constant(original.ErrOutputContext, nil),
	"ErrPartialCharset": scrigo.Constant(original.ErrPartialCharset, nil),
	"ErrPartialEscape": scrigo.Constant(original.ErrPartialEscape, nil),
	"ErrPredefinedEscaper": scrigo.Constant(original.ErrPredefinedEscaper, nil),
	"ErrRangeLoopReentry": scrigo.Constant(original.ErrRangeLoopReentry, nil),
	"ErrSlashAmbig": scrigo.Constant(original.ErrSlashAmbig, nil),
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
	"OK": scrigo.Constant(original.OK, nil),
	"ParseFiles": original.ParseFiles,
	"ParseGlob": original.ParseGlob,
	"Srcset": reflect.TypeOf(""),
	"Template": reflect.TypeOf(original.Template{}),
	"URL": reflect.TypeOf(""),
	"URLQueryEscaper": original.URLQueryEscaper,
}
