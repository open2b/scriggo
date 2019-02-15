// Go version: go1.11.5

package dsa

import original "crypto/dsa"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"ErrInvalidPublicKey": &original.ErrInvalidPublicKey,
	"GenerateKey": original.GenerateKey,
	"GenerateParameters": original.GenerateParameters,
	"ParameterSizes": reflect.TypeOf(original.ParameterSizes(int(0))),
	"Parameters": reflect.TypeOf(original.Parameters{}),
	"PrivateKey": reflect.TypeOf(original.PrivateKey{}),
	"PublicKey": reflect.TypeOf(original.PublicKey{}),
	"Sign": original.Sign,
	"Verify": original.Verify,
}
