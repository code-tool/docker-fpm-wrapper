package util

import (
	"testing"
	"bufio"
	"bytes"

	"github.com/stretchr/testify/assert"
)

func tf(t *testing.T, in string, out []byte) {
	r := bufio.NewReaderSize(bytes.NewBufferString(in), 16)

	line, err := ReadLine(r)
	assert.NoError(t, err)
	assert.Equal(t, out, line)
}

func TestReadLine(t *testing.T) {
	tf(t, "test\n", []byte("test\n"))
	tf(t, "test\r\n", []byte("test\n"))
	tf(t, "no line end", nil)
	tf(t, "test very long long line\n", nil)
}
