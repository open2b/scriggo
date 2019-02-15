// Go version: go1.11.5

package httptest

import "reflect"
import original "net/http/httptest"
import "scrigo"

var Package = scrigo.Package{
	"NewRecorder": original.NewRecorder,
	"NewRequest": original.NewRequest,
	"NewServer": original.NewServer,
	"NewTLSServer": original.NewTLSServer,
	"NewUnstartedServer": original.NewUnstartedServer,
	"ResponseRecorder": reflect.TypeOf(original.ResponseRecorder{}),
	"Server": reflect.TypeOf(original.Server{}),
}
