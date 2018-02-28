package util

import (
	"sync"
	"bufio"
	"bytes"
	"io"
)

type ReaderPool struct {
	sPool      sync.Pool
	nullReader io.Reader
}

func NewReaderPool(bufSize int) *ReaderPool {
	nullReader := bytes.NewReader(nil)

	return &ReaderPool{
		nullReader: nullReader,
		sPool: sync.Pool{
			New: func() interface{} {
				return bufio.NewReaderSize(nullReader, bufSize)
			},
		},
	}
}

func (rp *ReaderPool) Get(r io.Reader) *bufio.Reader {
	result := rp.sPool.Get().(*bufio.Reader)
	result.Reset(r)

	return result
}

func (rp *ReaderPool) Put(r *bufio.Reader) {
	r.Reset(rp.nullReader)
	rp.sPool.Put(r)
}
