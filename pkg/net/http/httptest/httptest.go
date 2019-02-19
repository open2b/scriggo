// Go version: go1.11.5

package httptest

import original "net/http/httptest"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"DefaultRemoteAddr": scrigo.Constant(original.DefaultRemoteAddr, nil),
	"NewRecorder": original.NewRecorder,
	"NewRequest": original.NewRequest,
	"NewServer": original.NewServer,
	"NewTLSServer": original.NewTLSServer,
	"NewUnstartedServer": original.NewUnstartedServer,
	"ResponseRecorder": reflect.TypeOf(original.ResponseRecorder{}),
	"Server": reflect.TypeOf(original.Server{}),
}
