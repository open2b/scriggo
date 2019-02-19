// Go version: go1.11.5

package binary

import original "encoding/binary"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"BigEndian": &original.BigEndian,
	"ByteOrder": reflect.TypeOf((*original.ByteOrder)(nil)).Elem(),
	"LittleEndian": &original.LittleEndian,
	"MaxVarintLen16": scrigo.Constant(original.MaxVarintLen16, nil),
	"MaxVarintLen32": scrigo.Constant(original.MaxVarintLen32, nil),
	"MaxVarintLen64": scrigo.Constant(original.MaxVarintLen64, nil),
	"PutUvarint": original.PutUvarint,
	"PutVarint": original.PutVarint,
	"Read": original.Read,
	"ReadUvarint": original.ReadUvarint,
	"ReadVarint": original.ReadVarint,
	"Size": original.Size,
	"Uvarint": original.Uvarint,
	"Varint": original.Varint,
	"Write": original.Write,
}
