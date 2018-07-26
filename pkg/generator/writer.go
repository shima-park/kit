package generator

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultFilePath   = "./"
	DefaultFileSuffix = "_gen"
)

var DefaultWriter = NewWriter()

type Writer struct {
	Writer  io.WriteCloser
	Options WriterOptions
}

func NewWriter(opts ...WriterOption) *Writer {
	var options WriterOptions
	for _, opt := range opts {
		opt(&options)
	}

	if options.FileName == "" {
		return &Writer{
			Options: options,
			Writer:  os.Stdout,
		}
	}

	if options.FileSuffix == "" {
		options.FileSuffix = DefaultFileSuffix
	}

	dir, filename := filepath.Split(options.FileName)
	newFilename := newFilenameWithSuffix(filename, options.FileSuffix)

	if options.FilePath == "" {
		if dir != "" {
			options.FilePath = dir
		} else {
			options.FilePath = DefaultFilePath
		}
	}

	writer, err := os.Create(newFilename)
	if err != nil {
		panic(err)
	}

	return &Writer{
		Options: options,
		Writer:  writer,
	}
}

func newFilenameWithSuffix(filename, fileSuffix string) string {
	extension := filepath.Ext(filename)
	name := filename[0 : len(filename)-len(extension)]
	if strings.HasSuffix(name, fileSuffix) {
		return filename
	}
	return name + fileSuffix + "." + extension
}

func (w Writer) Write(p []byte) (n int, err error) {
	return w.Writer.Write(p)
}

func (w Writer) Close() error {
	return w.Writer.Close()
}

type WriterOptions struct {
	FileName   string
	FileSuffix string
	FilePath   string
}

type WriterOption func(*WriterOptions)

func WithFileName(fn string) WriterOption {
	return func(o *WriterOptions) {
		o.FileName = fn
	}
}

func WithFileSuffix(fs string) WriterOption {
	return func(o *WriterOptions) {
		o.FileSuffix = fs
	}
}

func WithFilePath(fp string) WriterOption {
	return func(o *WriterOptions) {
		o.FilePath = fp
	}
}
