// Go version: go1.11.5

package sha512

import original "crypto/sha512"
import "scrigo"

var Package = scrigo.Package{
	"BlockSize": scrigo.Constant(original.BlockSize, nil),
	"New": original.New,
	"New384": original.New384,
	"New512_224": original.New512_224,
	"New512_256": original.New512_256,
	"Size": scrigo.Constant(original.Size, nil),
	"Size224": scrigo.Constant(original.Size224, nil),
	"Size256": scrigo.Constant(original.Size256, nil),
	"Size384": scrigo.Constant(original.Size384, nil),
	"Sum384": original.Sum384,
	"Sum512": original.Sum512,
	"Sum512_224": original.Sum512_224,
	"Sum512_256": original.Sum512_256,
}
