// Go version: go1.11.5

package sha1

import original "crypto/sha1"
import "scrigo"

var Package = scrigo.Package{
	"New": original.New,
	"Sum": original.Sum,
}
