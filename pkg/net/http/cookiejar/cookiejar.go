// Go version: go1.11.5

package cookiejar

import original "net/http/cookiejar"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Jar": reflect.TypeOf(original.Jar{}),
	"New": original.New,
	"Options": reflect.TypeOf(original.Options{}),
	"PublicSuffixList": reflect.TypeOf((*original.PublicSuffixList)(nil)).Elem(),
}
