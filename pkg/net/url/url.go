// Go version: go1.11.5

package url

import original "net/url"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Error": reflect.TypeOf(original.Error{}),
	"EscapeError": reflect.TypeOf(""),
	"InvalidHostError": reflect.TypeOf(""),
	"Parse": original.Parse,
	"ParseQuery": original.ParseQuery,
	"ParseRequestURI": original.ParseRequestURI,
	"PathEscape": original.PathEscape,
	"PathUnescape": original.PathUnescape,
	"QueryEscape": original.QueryEscape,
	"QueryUnescape": original.QueryUnescape,
	"URL": reflect.TypeOf(original.URL{}),
	"User": original.User,
	"UserPassword": original.UserPassword,
	"Userinfo": reflect.TypeOf(original.Userinfo{}),
	"Values": reflect.TypeOf((original.Values)(nil)),
}
