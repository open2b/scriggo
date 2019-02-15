// Go version: go1.11.5

package fcgi

import original "net/http/fcgi"
import "scrigo"

var Package = scrigo.Package{
	"ErrConnClosed": &original.ErrConnClosed,
	"ErrRequestAborted": &original.ErrRequestAborted,
	"ProcessEnv": original.ProcessEnv,
	"Serve": original.Serve,
}
