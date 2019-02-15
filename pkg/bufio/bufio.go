// Go version: go1.11.5

package bufio

import "reflect"
import original "bufio"
import "scrigo"

var Package = scrigo.Package{
	"ErrAdvanceTooFar": &original.ErrAdvanceTooFar,
	"ErrBufferFull": &original.ErrBufferFull,
	"ErrFinalToken": &original.ErrFinalToken,
	"ErrInvalidUnreadByte": &original.ErrInvalidUnreadByte,
	"ErrInvalidUnreadRune": &original.ErrInvalidUnreadRune,
	"ErrNegativeAdvance": &original.ErrNegativeAdvance,
	"ErrNegativeCount": &original.ErrNegativeCount,
	"ErrTooLong": &original.ErrTooLong,
	"NewReadWriter": original.NewReadWriter,
	"NewReader": original.NewReader,
	"NewReaderSize": original.NewReaderSize,
	"NewScanner": original.NewScanner,
	"NewWriter": original.NewWriter,
	"NewWriterSize": original.NewWriterSize,
	"ReadWriter": reflect.TypeOf(original.ReadWriter{}),
	"Reader": reflect.TypeOf(original.Reader{}),
	"ScanBytes": original.ScanBytes,
	"ScanLines": original.ScanLines,
	"ScanRunes": original.ScanRunes,
	"ScanWords": original.ScanWords,
	"Scanner": reflect.TypeOf(original.Scanner{}),
	"SplitFunc": reflect.TypeOf((original.SplitFunc)(nil)),
	"Writer": reflect.TypeOf(original.Writer{}),
}
