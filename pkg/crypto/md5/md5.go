// Go version: go1.11.5

package md5

import original "crypto/md5"
import "scrigo"

var Package = scrigo.Package{
	"BlockSize": scrigo.Constant(original.BlockSize, nil),
	"New": original.New,
	"Size": scrigo.Constant(original.Size, nil),
	"Sum": original.Sum,
}
