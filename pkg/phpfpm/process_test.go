package phpfpm

import (
	"testing"
	"bytes"

	"github.com/stretchr/testify/assert"
)

func TestNewProcess(t *testing.T) {
	stdout := bytes.NewBufferString("")
	stderr := bytes.NewBufferString("")

	p := NewProcess("echo", "configpath", stdout, stderr, "/tmp/sock", 0, "-n")
	assert.NoError(t, p.Start())

	errCh := make(chan error, 1)
	code := p.Wait(errCh)

	assert.Equal(t, 0, code)
	assert.Equal(t, "--nodaemonize --fpm-config configpath", stdout.String())
	assert.Equal(t, "", stderr.String())
}
