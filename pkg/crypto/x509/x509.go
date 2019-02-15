// Go version: go1.11.5

package x509

import original "crypto/x509"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"CertPool": reflect.TypeOf(original.CertPool{}),
	"Certificate": reflect.TypeOf(original.Certificate{}),
	"CertificateInvalidError": reflect.TypeOf(original.CertificateInvalidError{}),
	"CertificateRequest": reflect.TypeOf(original.CertificateRequest{}),
	"ConstraintViolationError": reflect.TypeOf(original.ConstraintViolationError{}),
	"CreateCertificate": original.CreateCertificate,
	"CreateCertificateRequest": original.CreateCertificateRequest,
	"DecryptPEMBlock": original.DecryptPEMBlock,
	"EncryptPEMBlock": original.EncryptPEMBlock,
	"ErrUnsupportedAlgorithm": &original.ErrUnsupportedAlgorithm,
	"ExtKeyUsage": reflect.TypeOf(original.ExtKeyUsage(int(0))),
	"HostnameError": reflect.TypeOf(original.HostnameError{}),
	"IncorrectPasswordError": &original.IncorrectPasswordError,
	"InsecureAlgorithmError": reflect.TypeOf(original.InsecureAlgorithmError(int(0))),
	"InvalidReason": reflect.TypeOf(original.InvalidReason(int(0))),
	"IsEncryptedPEMBlock": original.IsEncryptedPEMBlock,
	"KeyUsage": reflect.TypeOf(original.KeyUsage(int(0))),
	"MarshalECPrivateKey": original.MarshalECPrivateKey,
	"MarshalPKCS1PrivateKey": original.MarshalPKCS1PrivateKey,
	"MarshalPKCS1PublicKey": original.MarshalPKCS1PublicKey,
	"MarshalPKCS8PrivateKey": original.MarshalPKCS8PrivateKey,
	"MarshalPKIXPublicKey": original.MarshalPKIXPublicKey,
	"NewCertPool": original.NewCertPool,
	"PEMCipher": reflect.TypeOf(original.PEMCipher(int(0))),
	"ParseCRL": original.ParseCRL,
	"ParseCertificate": original.ParseCertificate,
	"ParseCertificateRequest": original.ParseCertificateRequest,
	"ParseCertificates": original.ParseCertificates,
	"ParseDERCRL": original.ParseDERCRL,
	"ParseECPrivateKey": original.ParseECPrivateKey,
	"ParsePKCS1PrivateKey": original.ParsePKCS1PrivateKey,
	"ParsePKCS1PublicKey": original.ParsePKCS1PublicKey,
	"ParsePKCS8PrivateKey": original.ParsePKCS8PrivateKey,
	"ParsePKIXPublicKey": original.ParsePKIXPublicKey,
	"PublicKeyAlgorithm": reflect.TypeOf(original.PublicKeyAlgorithm(int(0))),
	"SignatureAlgorithm": reflect.TypeOf(original.SignatureAlgorithm(int(0))),
	"SystemCertPool": original.SystemCertPool,
	"SystemRootsError": reflect.TypeOf(original.SystemRootsError{}),
	"UnhandledCriticalExtension": reflect.TypeOf(original.UnhandledCriticalExtension{}),
	"UnknownAuthorityError": reflect.TypeOf(original.UnknownAuthorityError{}),
	"VerifyOptions": reflect.TypeOf(original.VerifyOptions{}),
}
