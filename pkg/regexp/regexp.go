// Go version: go1.11.5

package regexp

import "scrigo"
import "reflect"
import original "regexp"

var Package = scrigo.Package{
	"Compile": original.Compile,
	"CompilePOSIX": original.CompilePOSIX,
	"Match": original.Match,
	"MatchReader": original.MatchReader,
	"MatchString": original.MatchString,
	"MustCompile": original.MustCompile,
	"MustCompilePOSIX": original.MustCompilePOSIX,
	"QuoteMeta": original.QuoteMeta,
	"Regexp": reflect.TypeOf(original.Regexp{}),
}
