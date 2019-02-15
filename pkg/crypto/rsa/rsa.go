// Go version: go1.11.5

package rsa

import "scrigo"
import "reflect"
import original "crypto/rsa"

var Package = scrigo.Package{
	"CRTValue": reflect.TypeOf(original.CRTValue{}),
	"DecryptOAEP": original.DecryptOAEP,
	"DecryptPKCS1v15": original.DecryptPKCS1v15,
	"DecryptPKCS1v15SessionKey": original.DecryptPKCS1v15SessionKey,
	"EncryptOAEP": original.EncryptOAEP,
	"EncryptPKCS1v15": original.EncryptPKCS1v15,
	"ErrDecryption": &original.ErrDecryption,
	"ErrMessageTooLong": &original.ErrMessageTooLong,
	"ErrVerification": &original.ErrVerification,
	"GenerateKey": original.GenerateKey,
	"GenerateMultiPrimeKey": original.GenerateMultiPrimeKey,
	"OAEPOptions": reflect.TypeOf(original.OAEPOptions{}),
	"PKCS1v15DecryptOptions": reflect.TypeOf(original.PKCS1v15DecryptOptions{}),
	"PSSOptions": reflect.TypeOf(original.PSSOptions{}),
	"PrecomputedValues": reflect.TypeOf(original.PrecomputedValues{}),
	"PrivateKey": reflect.TypeOf(original.PrivateKey{}),
	"PublicKey": reflect.TypeOf(original.PublicKey{}),
	"SignPKCS1v15": original.SignPKCS1v15,
	"SignPSS": original.SignPSS,
	"VerifyPKCS1v15": original.VerifyPKCS1v15,
	"VerifyPSS": original.VerifyPSS,
}
