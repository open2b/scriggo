// Go version: go1.11.5

package mail

import original "net/mail"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Address": reflect.TypeOf(original.Address{}),
	"AddressParser": reflect.TypeOf(original.AddressParser{}),
	"ErrHeaderNotPresent": &original.ErrHeaderNotPresent,
	"Header": reflect.TypeOf((original.Header)(nil)),
	"Message": reflect.TypeOf(original.Message{}),
	"ParseAddress": original.ParseAddress,
	"ParseAddressList": original.ParseAddressList,
	"ParseDate": original.ParseDate,
	"ReadMessage": original.ReadMessage,
}
