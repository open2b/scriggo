// Go version: go1.11.5

package httputil

import original "net/http/httputil"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"BufferPool": reflect.TypeOf((*original.BufferPool)(nil)).Elem(),
	"ClientConn": reflect.TypeOf(original.ClientConn{}),
	"DumpRequest": original.DumpRequest,
	"DumpRequestOut": original.DumpRequestOut,
	"DumpResponse": original.DumpResponse,
	"ErrClosed": &original.ErrClosed,
	"ErrLineTooLong": &original.ErrLineTooLong,
	"ErrPersistEOF": &original.ErrPersistEOF,
	"ErrPipeline": &original.ErrPipeline,
	"NewChunkedReader": original.NewChunkedReader,
	"NewChunkedWriter": original.NewChunkedWriter,
	"NewClientConn": original.NewClientConn,
	"NewProxyClientConn": original.NewProxyClientConn,
	"NewServerConn": original.NewServerConn,
	"NewSingleHostReverseProxy": original.NewSingleHostReverseProxy,
	"ReverseProxy": reflect.TypeOf(original.ReverseProxy{}),
	"ServerConn": reflect.TypeOf(original.ServerConn{}),
}
