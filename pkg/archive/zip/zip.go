// Go version: go1.11.5

package zip

import original "archive/zip"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Compressor": reflect.TypeOf((original.Compressor)(nil)),
	"Decompressor": reflect.TypeOf((original.Decompressor)(nil)),
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
	"Writer": reflect.TypeOf(original.Writer{}),
}
