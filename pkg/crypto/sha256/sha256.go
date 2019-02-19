// Go version: go1.11.5

package sha256

import original "crypto/sha256"
import "scrigo"

var Package = scrigo.Package{
	"BlockSize": scrigo.Constant(original.BlockSize, nil),
	"New": original.New,
	"New224": original.New224,
	"Size": scrigo.Constant(original.Size, nil),
	"Size224": scrigo.Constant(original.Size224, nil),
	"Sum224": original.Sum224,
	"Sum256": original.Sum256,
}
