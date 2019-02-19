// Go version: go1.11.5

package adler32

import original "hash/adler32"
import "scrigo"

var Package = scrigo.Package{
	"Checksum": original.Checksum,
	"New": original.New,
	"Size": scrigo.Constant(original.Size, nil),
}
