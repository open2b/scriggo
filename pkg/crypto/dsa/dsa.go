// Go version: go1.11.5

package dsa

import original "crypto/dsa"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"ErrInvalidPublicKey": &original.ErrInvalidPublicKey,
	"GenerateKey": original.GenerateKey,
	"GenerateParameters": original.GenerateParameters,
	"L1024N160": scrigo.Constant(original.L1024N160, nil),
	"L2048N224": scrigo.Constant(original.L2048N224, nil),
	"L2048N256": scrigo.Constant(original.L2048N256, nil),
	"L3072N256": scrigo.Constant(original.L3072N256, nil),
	"ParameterSizes": reflect.TypeOf(original.ParameterSizes(int(0))),
	"Parameters": reflect.TypeOf(original.Parameters{}),
	"PrivateKey": reflect.TypeOf(original.PrivateKey{}),
	"PublicKey": reflect.TypeOf(original.PublicKey{}),
	"Sign": original.Sign,
	"Verify": original.Verify,
}
