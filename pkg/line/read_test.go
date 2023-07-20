package line

import (
	"bufio"
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func tf(t *testing.T, in string, out []byte, expectedErr error) {
	r := bufio.NewReaderSize(bytes.NewBufferString(in), 16)

	line, err := ReadOne(r, false)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, out, line)
}

func TestReadLine(t *testing.T) {
	tf(t, "test\n", []byte("test\n"), nil)
	tf(t, "no line end", nil, io.EOF)
	tf(t, "test very long long line\n", nil, io.EOF)
	tf(t, "test very long long line\nSecond line\n", []byte("Second line\n"), nil)
}
