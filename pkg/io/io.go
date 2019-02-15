// Go version: go1.11.5

package io

import original "io"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"ByteReader": reflect.TypeOf((*original.ByteReader)(nil)).Elem(),
	"ByteScanner": reflect.TypeOf((*original.ByteScanner)(nil)).Elem(),
	"ByteWriter": reflect.TypeOf((*original.ByteWriter)(nil)).Elem(),
	"Closer": reflect.TypeOf((*original.Closer)(nil)).Elem(),
	"Copy": original.Copy,
	"CopyBuffer": original.CopyBuffer,
	"CopyN": original.CopyN,
	"EOF": &original.EOF,
	"ErrClosedPipe": &original.ErrClosedPipe,
	"ErrNoProgress": &original.ErrNoProgress,
	"ErrShortBuffer": &original.ErrShortBuffer,
	"ErrShortWrite": &original.ErrShortWrite,
	"ErrUnexpectedEOF": &original.ErrUnexpectedEOF,
	"LimitReader": original.LimitReader,
	"LimitedReader": reflect.TypeOf(original.LimitedReader{}),
	"MultiReader": original.MultiReader,
	"MultiWriter": original.MultiWriter,
	"NewSectionReader": original.NewSectionReader,
	"Pipe": original.Pipe,
	"PipeReader": reflect.TypeOf(original.PipeReader{}),
	"PipeWriter": reflect.TypeOf(original.PipeWriter{}),
	"ReadAtLeast": original.ReadAtLeast,
	"ReadCloser": reflect.TypeOf((*original.ReadCloser)(nil)).Elem(),
	"ReadFull": original.ReadFull,
	"ReadSeeker": reflect.TypeOf((*original.ReadSeeker)(nil)).Elem(),
	"ReadWriteCloser": reflect.TypeOf((*original.ReadWriteCloser)(nil)).Elem(),
	"ReadWriteSeeker": reflect.TypeOf((*original.ReadWriteSeeker)(nil)).Elem(),
	"ReadWriter": reflect.TypeOf((*original.ReadWriter)(nil)).Elem(),
	"Reader": reflect.TypeOf((*original.Reader)(nil)).Elem(),
	"ReaderAt": reflect.TypeOf((*original.ReaderAt)(nil)).Elem(),
	"ReaderFrom": reflect.TypeOf((*original.ReaderFrom)(nil)).Elem(),
	"RuneReader": reflect.TypeOf((*original.RuneReader)(nil)).Elem(),
	"RuneScanner": reflect.TypeOf((*original.RuneScanner)(nil)).Elem(),
	"SectionReader": reflect.TypeOf(original.SectionReader{}),
	"Seeker": reflect.TypeOf((*original.Seeker)(nil)).Elem(),
	"TeeReader": original.TeeReader,
	"WriteCloser": reflect.TypeOf((*original.WriteCloser)(nil)).Elem(),
	"WriteSeeker": reflect.TypeOf((*original.WriteSeeker)(nil)).Elem(),
	"WriteString": original.WriteString,
	"Writer": reflect.TypeOf((*original.Writer)(nil)).Elem(),
	"WriterAt": reflect.TypeOf((*original.WriterAt)(nil)).Elem(),
	"WriterTo": reflect.TypeOf((*original.WriterTo)(nil)).Elem(),
}
