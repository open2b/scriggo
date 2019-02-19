// Go version: go1.11.5

package des

import original "crypto/des"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"BlockSize": scrigo.Constant(original.BlockSize, nil),
	"KeySizeError": reflect.TypeOf(original.KeySizeError(int(0))),
	"NewCipher": original.NewCipher,
	"NewTripleDESCipher": original.NewTripleDESCipher,
}
