package util

import (
	"testing"
	"bufio"
	"bytes"

	"github.com/stretchr/testify/assert"
	"io"
)

func tf(t *testing.T, in string, out []byte, expectedErr error) {
	r := bufio.NewReaderSize(bytes.NewBufferString(in), 16)

	line, err := ReadLine(r)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, out, line)
}

func TestReadLine(t *testing.T) {
	tf(t, "test\n", []byte("test\n"), nil)
	tf(t, "test\r\n", []byte("test\n"), nil)
	tf(t, "no line end", nil, io.EOF)
	tf(t, "test very long long line\n", nil, nil)
}
