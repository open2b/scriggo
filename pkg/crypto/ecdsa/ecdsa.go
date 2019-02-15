// Go version: go1.11.5

package ecdsa

import "reflect"
import original "crypto/ecdsa"
import "scrigo"

var Package = scrigo.Package{
	"GenerateKey": original.GenerateKey,
	"PrivateKey": reflect.TypeOf(original.PrivateKey{}),
	"PublicKey": reflect.TypeOf(original.PublicKey{}),
	"Sign": original.Sign,
	"Verify": original.Verify,
}
