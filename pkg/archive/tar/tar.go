// Go version: go1.11.5

package tar

import original "archive/tar"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"ErrFieldTooLong": &original.ErrFieldTooLong,
	"ErrHeader": &original.ErrHeader,
	"ErrWriteAfterClose": &original.ErrWriteAfterClose,
	"ErrWriteTooLong": &original.ErrWriteTooLong,
	"FileInfoHeader": original.FileInfoHeader,
	"Format": reflect.TypeOf(original.Format(int(0))),
	"FormatGNU": scrigo.Constant(original.FormatGNU, nil),
	"FormatPAX": scrigo.Constant(original.FormatPAX, nil),
	"FormatUSTAR": scrigo.Constant(original.FormatUSTAR, nil),
	"FormatUnknown": scrigo.Constant(original.FormatUnknown, nil),
	"Header": reflect.TypeOf(original.Header{}),
	"NewReader": original.NewReader,
	"NewWriter": original.NewWriter,
	"Reader": reflect.TypeOf(original.Reader{}),
	"TypeBlock": scrigo.Constant(original.TypeBlock, nil),
	"TypeChar": scrigo.Constant(original.TypeChar, nil),
	"TypeCont": scrigo.Constant(original.TypeCont, nil),
	"TypeDir": scrigo.Constant(original.TypeDir, nil),
	"TypeFifo": scrigo.Constant(original.TypeFifo, nil),
	"TypeGNULongLink": scrigo.Constant(original.TypeGNULongLink, nil),
	"TypeGNULongName": scrigo.Constant(original.TypeGNULongName, nil),
	"TypeGNUSparse": scrigo.Constant(original.TypeGNUSparse, nil),
	"TypeLink": scrigo.Constant(original.TypeLink, nil),
	"TypeReg": scrigo.Constant(original.TypeReg, nil),
	"TypeRegA": scrigo.Constant(original.TypeRegA, nil),
	"TypeSymlink": scrigo.Constant(original.TypeSymlink, nil),
	"TypeXGlobalHeader": scrigo.Constant(original.TypeXGlobalHeader, nil),
	"TypeXHeader": scrigo.Constant(original.TypeXHeader, nil),
	"Writer": reflect.TypeOf(original.Writer{}),
}
