// Go version: go1.11.5

package rpc

import "scrigo"
import "reflect"
import original "net/rpc"

var Package = scrigo.Package{
	"Accept": original.Accept,
	"Call": reflect.TypeOf(original.Call{}),
	"Client": reflect.TypeOf(original.Client{}),
	"ClientCodec": reflect.TypeOf((*original.ClientCodec)(nil)).Elem(),
	"DefaultServer": &original.DefaultServer,
	"Dial": original.Dial,
	"DialHTTP": original.DialHTTP,
	"DialHTTPPath": original.DialHTTPPath,
	"ErrShutdown": &original.ErrShutdown,
	"HandleHTTP": original.HandleHTTP,
	"NewClient": original.NewClient,
	"NewClientWithCodec": original.NewClientWithCodec,
	"NewServer": original.NewServer,
	"Register": original.Register,
	"RegisterName": original.RegisterName,
	"Request": reflect.TypeOf(original.Request{}),
	"Response": reflect.TypeOf(original.Response{}),
	"ServeCodec": original.ServeCodec,
	"ServeConn": original.ServeConn,
	"ServeRequest": original.ServeRequest,
	"Server": reflect.TypeOf(original.Server{}),
	"ServerCodec": reflect.TypeOf((*original.ServerCodec)(nil)).Elem(),
	"ServerError": reflect.TypeOf(""),
}
