// Go version: go1.11.5

package httptrace

import original "net/http/httptrace"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"ClientTrace": reflect.TypeOf(original.ClientTrace{}),
	"ContextClientTrace": original.ContextClientTrace,
	"DNSDoneInfo": reflect.TypeOf(original.DNSDoneInfo{}),
	"DNSStartInfo": reflect.TypeOf(original.DNSStartInfo{}),
	"GotConnInfo": reflect.TypeOf(original.GotConnInfo{}),
	"WithClientTrace": original.WithClientTrace,
	"WroteRequestInfo": reflect.TypeOf(original.WroteRequestInfo{}),
}
