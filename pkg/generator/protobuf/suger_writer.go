package protobuf

import (
	"fmt"
	"io"
)

type SugerWriter struct {
	io.Writer
}

func NewSugerWriter(w io.Writer) *SugerWriter {
	return &SugerWriter{
		w,
	}
}

func (w *SugerWriter) P(format string, v ...interface{}) {
	if len(v) > 0 {
		w.Write([]byte(fmt.Sprintf(format, v...)))
		return
	}

	if len(v) == 0 && format == "" {
		format = "\n"
	}
	w.Write([]byte(format))
}
