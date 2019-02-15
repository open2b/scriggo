// Go version: go1.11.5

package sha512

import original "crypto/sha512"
import "scrigo"

var Package = scrigo.Package{
	"New": original.New,
	"New384": original.New384,
	"New512_224": original.New512_224,
	"New512_256": original.New512_256,
	"Sum384": original.Sum384,
	"Sum512": original.Sum512,
	"Sum512_224": original.Sum512_224,
	"Sum512_256": original.Sum512_256,
}
