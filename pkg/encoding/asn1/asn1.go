// Go version: go1.11.5

package asn1

import "reflect"
import original "encoding/asn1"
import "scrigo"

var Package = scrigo.Package{
	"BitString": reflect.TypeOf(original.BitString{}),
	"ClassApplication": scrigo.Constant(original.ClassApplication, nil),
	"ClassContextSpecific": scrigo.Constant(original.ClassContextSpecific, nil),
	"ClassPrivate": scrigo.Constant(original.ClassPrivate, nil),
	"ClassUniversal": scrigo.Constant(original.ClassUniversal, nil),
	"Enumerated": reflect.TypeOf(original.Enumerated(int(0))),
	"Flag": reflect.TypeOf(false),
	"Marshal": original.Marshal,
	"MarshalWithParams": original.MarshalWithParams,
	"NullBytes": &original.NullBytes,
	"NullRawValue": &original.NullRawValue,
	"ObjectIdentifier": reflect.TypeOf((original.ObjectIdentifier)(nil)),
	"RawContent": reflect.TypeOf((original.RawContent)(nil)),
	"RawValue": reflect.TypeOf(original.RawValue{}),
	"StructuralError": reflect.TypeOf(original.StructuralError{}),
	"SyntaxError": reflect.TypeOf(original.SyntaxError{}),
	"TagBitString": scrigo.Constant(original.TagBitString, nil),
	"TagBoolean": scrigo.Constant(original.TagBoolean, nil),
	"TagEnum": scrigo.Constant(original.TagEnum, nil),
	"TagGeneralString": scrigo.Constant(original.TagGeneralString, nil),
	"TagGeneralizedTime": scrigo.Constant(original.TagGeneralizedTime, nil),
	"TagIA5String": scrigo.Constant(original.TagIA5String, nil),
	"TagInteger": scrigo.Constant(original.TagInteger, nil),
	"TagNull": scrigo.Constant(original.TagNull, nil),
	"TagNumericString": scrigo.Constant(original.TagNumericString, nil),
	"TagOID": scrigo.Constant(original.TagOID, nil),
	"TagOctetString": scrigo.Constant(original.TagOctetString, nil),
	"TagPrintableString": scrigo.Constant(original.TagPrintableString, nil),
	"TagSequence": scrigo.Constant(original.TagSequence, nil),
	"TagSet": scrigo.Constant(original.TagSet, nil),
	"TagT61String": scrigo.Constant(original.TagT61String, nil),
	"TagUTCTime": scrigo.Constant(original.TagUTCTime, nil),
	"TagUTF8String": scrigo.Constant(original.TagUTF8String, nil),
	"Unmarshal": original.Unmarshal,
	"UnmarshalWithParams": original.UnmarshalWithParams,
}
