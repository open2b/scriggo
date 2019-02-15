// Go version: go1.11.5

package crc32

import original "hash/crc32"
import "scrigo"

var Package = scrigo.Package{
	"Checksum": original.Checksum,
	"ChecksumIEEE": original.ChecksumIEEE,
	"IEEETable": &original.IEEETable,
	"MakeTable": original.MakeTable,
	"New": original.New,
	"NewIEEE": original.NewIEEE,
	"Update": original.Update,
}
