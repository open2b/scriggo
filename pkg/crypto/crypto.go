// Go version: go1.11.5

package crypto

import original "crypto"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Decrypter": reflect.TypeOf((*original.Decrypter)(nil)).Elem(),
	"DecrypterOpts": reflect.TypeOf((*original.DecrypterOpts)(nil)).Elem(),
	"Hash": reflect.TypeOf(original.Hash(uint(0))),
	"PrivateKey": reflect.TypeOf((*original.PrivateKey)(nil)).Elem(),
	"PublicKey": reflect.TypeOf((*original.PublicKey)(nil)).Elem(),
	"RegisterHash": original.RegisterHash,
	"Signer": reflect.TypeOf((*original.Signer)(nil)).Elem(),
	"SignerOpts": reflect.TypeOf((*original.SignerOpts)(nil)).Elem(),
}
