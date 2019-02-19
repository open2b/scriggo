// Go version: go1.11.5

package zip

import "reflect"
import original "archive/zip"
import "scrigo"

var Package = scrigo.Package{
	"Compressor": reflect.TypeOf((original.Compressor)(nil)),
	"Decompressor": reflect.TypeOf((original.Decompressor)(nil)),
	"Deflate": scrigo.Constant(original.Deflate, nil),
	"ErrAlgorithm": &original.ErrAlgorithm,
	"ErrChecksum": &original.ErrChecksum,
	"ErrFormat": &original.ErrFormat,
	"File": reflect.TypeOf(original.File{}),
	"FileHeader": reflect.TypeOf(original.FileHeader{}),
	"FileInfoHeader": original.FileInfoHeader,
	"NewReader": original.NewReader,
	"NewWriter": original.NewWriter,
	"OpenReader": original.OpenReader,
	"ReadCloser": reflect.TypeOf(original.ReadCloser{}),
	"Reader": reflect.TypeOf(original.Reader{}),
	"RegisterCompressor": original.RegisterCompressor,
	"RegisterDecompressor": original.RegisterDecompressor,
	"Store": scrigo.Constant(original.Store, nil),
	"Writer": reflect.TypeOf(original.Writer{}),
}
