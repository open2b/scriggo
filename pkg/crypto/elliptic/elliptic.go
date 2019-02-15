// Go version: go1.11.5

package elliptic

import original "crypto/elliptic"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Curve": reflect.TypeOf((*original.Curve)(nil)).Elem(),
	"CurveParams": reflect.TypeOf(original.CurveParams{}),
	"GenerateKey": original.GenerateKey,
	"Marshal": original.Marshal,
	"P224": original.P224,
	"P256": original.P256,
	"P384": original.P384,
	"P521": original.P521,
	"Unmarshal": original.Unmarshal,
}
