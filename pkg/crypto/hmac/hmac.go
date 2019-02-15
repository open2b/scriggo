// Go version: go1.11.5

package hmac

import original "crypto/hmac"
import "scrigo"

var Package = scrigo.Package{
	"Equal": original.Equal,
	"New": original.New,
}
