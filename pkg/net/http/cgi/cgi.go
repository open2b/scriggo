// Go version: go1.11.5

package cgi

import original "net/http/cgi"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Handler": reflect.TypeOf(original.Handler{}),
	"Request": original.Request,
	"RequestFromMap": original.RequestFromMap,
	"Serve": original.Serve,
}
