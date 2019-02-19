// Go version: go1.11.5

package textproto

import original "net/textproto"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"CanonicalMIMEHeaderKey": original.CanonicalMIMEHeaderKey,
	"Conn": reflect.TypeOf(original.Conn{}),
	"Dial": original.Dial,
	"Error": reflect.TypeOf(original.Error{}),
	"MIMEHeader": reflect.TypeOf((original.MIMEHeader)(nil)),
	"NewConn": original.NewConn,
	"NewReader": original.NewReader,
	"NewWriter": original.NewWriter,
	"Pipeline": reflect.TypeOf(original.Pipeline{}),
	"ProtocolError": reflect.TypeOf(""),
	"Reader": reflect.TypeOf(original.Reader{}),
	"TrimBytes": original.TrimBytes,
	"TrimString": original.TrimString,
	"Writer": reflect.TypeOf(original.Writer{}),
}
