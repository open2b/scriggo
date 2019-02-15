// Go version: go1.11.5

package sha256

import original "crypto/sha256"
import "scrigo"

var Package = scrigo.Package{
	"New": original.New,
	"New224": original.New224,
	"Sum224": original.Sum224,
	"Sum256": original.Sum256,
}
