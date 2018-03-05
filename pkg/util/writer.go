package util

import (
	"io"
	"sync"
)

type syncWriter struct {
	w io.Writer
	l sync.Mutex
}

func NewSyncWriter(w io.Writer) io.Writer {
	return &syncWriter{w, sync.Mutex{}}
}

func (sw *syncWriter) Write(p []byte) (n int, err error) {
	sw.l.Lock()
	n, err = sw.w.Write(p)
	sw.l.Unlock()

	return
}
