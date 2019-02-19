// Go version: go1.11.5

package crc32

import original "hash/crc32"
import "scrigo"

var Package = scrigo.Package{
	"Castagnoli": scrigo.Constant(original.Castagnoli, nil),
	"Checksum": original.Checksum,
	"ChecksumIEEE": original.ChecksumIEEE,
	"IEEE": scrigo.Constant(original.IEEE, nil),
	"IEEETable": &original.IEEETable,
	"Koopman": scrigo.Constant(original.Koopman, nil),
	"MakeTable": original.MakeTable,
	"New": original.New,
	"NewIEEE": original.NewIEEE,
	"Size": scrigo.Constant(original.Size, nil),
	"Update": original.Update,
}
