package phpfpm

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewProcess(t *testing.T) {
	stdout := bytes.NewBufferString("")
	stderr := bytes.NewBufferString("")

	p := NewProcess(zap.NewNop(), "echo", "configpath", stdout, stderr, 0, []string{}, "-n")
	assert.NoError(t, p.Start())

	errCh := make(chan error, 1)
	code := p.Wait(errCh)

	assert.Equal(t, 0, code)
	assert.Equal(t, "--nodaemonize --fpm-config configpath", stdout.String())
	assert.Equal(t, "", stderr.String())
}
