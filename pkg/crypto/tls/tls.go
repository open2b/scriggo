// Go version: go1.11.5

package tls

import original "crypto/tls"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Certificate": reflect.TypeOf(original.Certificate{}),
	"CertificateRequestInfo": reflect.TypeOf(original.CertificateRequestInfo{}),
	"Client": original.Client,
	"ClientAuthType": reflect.TypeOf(original.ClientAuthType(int(0))),
	"ClientHelloInfo": reflect.TypeOf(original.ClientHelloInfo{}),
	"ClientSessionCache": reflect.TypeOf((*original.ClientSessionCache)(nil)).Elem(),
	"ClientSessionState": reflect.TypeOf(original.ClientSessionState{}),
	"Config": reflect.TypeOf(original.Config{}),
	"Conn": reflect.TypeOf(original.Conn{}),
	"ConnectionState": reflect.TypeOf(original.ConnectionState{}),
	"CurveID": reflect.TypeOf(original.CurveID(uint16(0))),
	"Dial": original.Dial,
	"DialWithDialer": original.DialWithDialer,
	"Listen": original.Listen,
	"LoadX509KeyPair": original.LoadX509KeyPair,
	"NewLRUClientSessionCache": original.NewLRUClientSessionCache,
	"NewListener": original.NewListener,
	"RecordHeaderError": reflect.TypeOf(original.RecordHeaderError{}),
	"RenegotiationSupport": reflect.TypeOf(original.RenegotiationSupport(int(0))),
	"Server": original.Server,
	"SignatureScheme": reflect.TypeOf(original.SignatureScheme(uint16(0))),
	"X509KeyPair": original.X509KeyPair,
}
