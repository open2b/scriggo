// Go version: go1.11.5

package jsonrpc

import original "net/rpc/jsonrpc"
import "scrigo"

var Package = scrigo.Package{
	"Dial": original.Dial,
	"NewClient": original.NewClient,
	"NewClientCodec": original.NewClientCodec,
	"NewServerCodec": original.NewServerCodec,
	"ServeConn": original.ServeConn,
}
