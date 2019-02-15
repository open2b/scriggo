// Go version: go1.11.5

package rc4

import original "crypto/rc4"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Cipher": reflect.TypeOf(original.Cipher{}),
	"KeySizeError": reflect.TypeOf(original.KeySizeError(int(0))),
	"NewCipher": original.NewCipher,
}
