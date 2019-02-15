// Go version: go1.11.5

package crc64

import original "hash/crc64"
import "scrigo"

var Package = scrigo.Package{
	"Checksum": original.Checksum,
	"MakeTable": original.MakeTable,
	"New": original.New,
	"Update": original.Update,
}
