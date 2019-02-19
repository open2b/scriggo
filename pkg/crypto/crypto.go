// Go version: go1.11.5

package crypto

import original "crypto"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"BLAKE2b_256": scrigo.Constant(original.BLAKE2b_256, nil),
	"BLAKE2b_384": scrigo.Constant(original.BLAKE2b_384, nil),
	"BLAKE2b_512": scrigo.Constant(original.BLAKE2b_512, nil),
	"BLAKE2s_256": scrigo.Constant(original.BLAKE2s_256, nil),
	"Decrypter": reflect.TypeOf((*original.Decrypter)(nil)).Elem(),
	"DecrypterOpts": reflect.TypeOf((*original.DecrypterOpts)(nil)).Elem(),
	"Hash": reflect.TypeOf(original.Hash(uint(0))),
	"MD4": scrigo.Constant(original.MD4, nil),
	"MD5": scrigo.Constant(original.MD5, nil),
	"MD5SHA1": scrigo.Constant(original.MD5SHA1, nil),
	"PrivateKey": reflect.TypeOf((*original.PrivateKey)(nil)).Elem(),
	"PublicKey": reflect.TypeOf((*original.PublicKey)(nil)).Elem(),
	"RIPEMD160": scrigo.Constant(original.RIPEMD160, nil),
	"RegisterHash": original.RegisterHash,
	"SHA1": scrigo.Constant(original.SHA1, nil),
	"SHA224": scrigo.Constant(original.SHA224, nil),
	"SHA256": scrigo.Constant(original.SHA256, nil),
	"SHA384": scrigo.Constant(original.SHA384, nil),
	"SHA3_224": scrigo.Constant(original.SHA3_224, nil),
	"SHA3_256": scrigo.Constant(original.SHA3_256, nil),
	"SHA3_384": scrigo.Constant(original.SHA3_384, nil),
	"SHA3_512": scrigo.Constant(original.SHA3_512, nil),
	"SHA512": scrigo.Constant(original.SHA512, nil),
	"SHA512_224": scrigo.Constant(original.SHA512_224, nil),
	"SHA512_256": scrigo.Constant(original.SHA512_256, nil),
	"Signer": reflect.TypeOf((*original.Signer)(nil)).Elem(),
	"SignerOpts": reflect.TypeOf((*original.SignerOpts)(nil)).Elem(),
}
