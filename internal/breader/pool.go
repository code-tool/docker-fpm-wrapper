package breader

import (
	"bufio"
	"bytes"
	"io"
	"sync"
)

type Pool struct {
	sPool      sync.Pool
	nullReader io.Reader
}

func NewPool(bufSize int) *Pool {
	nullReader := bytes.NewReader(nil)

	return &Pool{
		nullReader: nullReader,
		sPool: sync.Pool{New: func() any {
			return bufio.NewReaderSize(nullReader, bufSize)
		}},
	}
}

func (rp *Pool) Get(r io.Reader) *bufio.Reader {
	result := rp.sPool.Get().(*bufio.Reader)
	result.Reset(r)

	return result
}

func (rp *Pool) Put(r *bufio.Reader) {
	r.Reset(rp.nullReader)
	rp.sPool.Put(r)
}
