// Go version: go1.11.5

package ecdsa

import original "crypto/ecdsa"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"GenerateKey": original.GenerateKey,
	"PrivateKey": reflect.TypeOf(original.PrivateKey{}),
	"PublicKey": reflect.TypeOf(original.PublicKey{}),
	"Sign": original.Sign,
	"Verify": original.Verify,
}
