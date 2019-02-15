// Go version: go1.11.5

package cipher

import original "crypto/cipher"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"AEAD": reflect.TypeOf((*original.AEAD)(nil)).Elem(),
	"Block": reflect.TypeOf((*original.Block)(nil)).Elem(),
	"BlockMode": reflect.TypeOf((*original.BlockMode)(nil)).Elem(),
	"NewCBCDecrypter": original.NewCBCDecrypter,
	"NewCBCEncrypter": original.NewCBCEncrypter,
	"NewCFBDecrypter": original.NewCFBDecrypter,
	"NewCFBEncrypter": original.NewCFBEncrypter,
	"NewCTR": original.NewCTR,
	"NewGCM": original.NewGCM,
	"NewGCMWithNonceSize": original.NewGCMWithNonceSize,
	"NewGCMWithTagSize": original.NewGCMWithTagSize,
	"NewOFB": original.NewOFB,
	"Stream": reflect.TypeOf((*original.Stream)(nil)).Elem(),
	"StreamReader": reflect.TypeOf(original.StreamReader{}),
	"StreamWriter": reflect.TypeOf(original.StreamWriter{}),
}
