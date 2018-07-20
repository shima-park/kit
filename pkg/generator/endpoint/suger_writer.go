package endpoint

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
	format += "\n"
	if len(v) > 0 {
		w.Write([]byte(fmt.Sprintf(format, v...)))
		return
	}
	w.Write([]byte(format))

}
