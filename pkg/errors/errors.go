// Go version: go1.11.5

package errors

import original "errors"
import "scrigo"

var Package = scrigo.Package{
	"New": original.New,
}
