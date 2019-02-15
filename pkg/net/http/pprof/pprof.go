// Go version: go1.11.5

package pprof

import original "net/http/pprof"
import "scrigo"

var Package = scrigo.Package{
	"Cmdline": original.Cmdline,
	"Handler": original.Handler,
	"Index": original.Index,
	"Profile": original.Profile,
	"Symbol": original.Symbol,
	"Trace": original.Trace,
}
