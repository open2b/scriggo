// Go version: go1.11.5

package smtp

import original "net/smtp"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Auth": reflect.TypeOf((*original.Auth)(nil)).Elem(),
	"CRAMMD5Auth": original.CRAMMD5Auth,
	"Client": reflect.TypeOf(original.Client{}),
	"Dial": original.Dial,
	"NewClient": original.NewClient,
	"PlainAuth": original.PlainAuth,
	"SendMail": original.SendMail,
	"ServerInfo": reflect.TypeOf(original.ServerInfo{}),
}
