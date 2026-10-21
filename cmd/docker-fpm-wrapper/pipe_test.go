package main

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateFIFOByPath changes the umask of the process, so it does not run in parallel.
func TestCreateFIFOByPath(t *testing.T) {
	tests := []struct {
		name  string
		umask int
		perm  os.FileMode
	}{
		{name: "wrapper pipe with default umask", umask: 0o022, perm: wrapperPipePerm},
		{name: "wrapper pipe with strict umask", umask: 0o077, perm: wrapperPipePerm},
		{name: "fpm log pipe with strict umask", umask: 0o077, perm: fpmLogPipePerm},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldUmask := syscall.Umask(tt.umask)
			t.Cleanup(func() { syscall.Umask(oldUmask) })

			fPath := filepath.Join(t.TempDir(), "pipe")
			f, err := createFIFOByPath(fPath, tt.perm)
			require.NoError(t, err)
			t.Cleanup(func() { _ = f.Close() })

			info, err := os.Stat(fPath)
			require.NoError(t, err)
			assert.Equal(t, os.ModeNamedPipe|tt.perm, info.Mode())
		})
	}

	t.Run("existing file is replaced", func(t *testing.T) {
		fPath := filepath.Join(t.TempDir(), "pipe")
		require.NoError(t, os.WriteFile(fPath, []byte("data"), 0o600))

		f, err := createFIFOByPath(fPath, wrapperPipePerm)
		require.NoError(t, err)
		t.Cleanup(func() { _ = f.Close() })

		info, err := os.Stat(fPath)
		require.NoError(t, err)
		assert.Equal(t, os.ModeNamedPipe|wrapperPipePerm, info.Mode())
	})

	t.Run("missing directory is an error", func(t *testing.T) {
		_, err := createFIFOByPath(filepath.Join(t.TempDir(), "missing", "pipe"), wrapperPipePerm)
		assert.ErrorContains(t, err, "can't create linux pipe")
	})
}
