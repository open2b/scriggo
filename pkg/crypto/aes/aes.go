// Go version: go1.11.5

package aes

import original "crypto/aes"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"KeySizeError": reflect.TypeOf(original.KeySizeError(int(0))),
	"NewCipher": original.NewCipher,
}
