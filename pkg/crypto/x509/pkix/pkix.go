// Go version: go1.11.5

package pkix

import "scrigo"
import "reflect"
import original "crypto/x509/pkix"

var Package = scrigo.Package{
	"AlgorithmIdentifier": reflect.TypeOf(original.AlgorithmIdentifier{}),
	"AttributeTypeAndValue": reflect.TypeOf(original.AttributeTypeAndValue{}),
	"AttributeTypeAndValueSET": reflect.TypeOf(original.AttributeTypeAndValueSET{}),
	"CertificateList": reflect.TypeOf(original.CertificateList{}),
	"Extension": reflect.TypeOf(original.Extension{}),
	"Name": reflect.TypeOf(original.Name{}),
	"RDNSequence": reflect.TypeOf((original.RDNSequence)(nil)),
	"RelativeDistinguishedNameSET": reflect.TypeOf((original.RelativeDistinguishedNameSET)(nil)),
	"RevokedCertificate": reflect.TypeOf(original.RevokedCertificate{}),
	"TBSCertificateList": reflect.TypeOf(original.TBSCertificateList{}),
}
