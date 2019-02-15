// Go version: go1.11.5

package des

import "scrigo"
import "reflect"
import original "crypto/des"

var Package = scrigo.Package{
	"KeySizeError": reflect.TypeOf(original.KeySizeError(int(0))),
	"NewCipher": original.NewCipher,
	"NewTripleDESCipher": original.NewTripleDESCipher,
}
