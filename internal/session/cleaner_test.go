package session

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// testMaster is the php-fpm master for the tests: the test process itself. Then the files
// that the test opens are open files of the master.
type testMaster struct{}

func (testMaster) Pid() int {
	return os.Getpid()
}

func writeSessionFile(t *testing.T, path string, modTime time.Time) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte("data"), 0o600))
	require.NoError(t, os.Chtimes(path, modTime, modTime))
}

func TestCleaner_Cleanup(t *testing.T) {
	const lifetime = time.Hour
	now := time.Now()
	expired := now.Add(-2 * lifetime)

	tests := []struct {
		name       string
		file       string
		modTime    time.Time
		open       bool
		wantExists bool
	}{
		{name: "expired session is removed", file: "sess_expired", modTime: expired},
		{name: "live session stays", file: "sess_live", modTime: now.Add(-time.Minute), wantExists: true},
		{name: "expired session in a subdirectory is removed", file: "a/b/sess_nested", modTime: expired},
		{name: "expired file without the sess_ prefix stays", file: "other", modTime: expired, wantExists: true},
		{name: "expired open session stays", file: "sess_open", modTime: expired, open: true, wantExists: true},
	}

	dir := t.TempDir()
	for _, tt := range tests {
		path := filepath.Join(dir, tt.file)
		writeSessionFile(t, path, tt.modTime)

		if tt.open {
			f, err := os.Open(path)
			require.NoError(t, err)
			t.Cleanup(func() { _ = f.Close() })
		}
	}

	cleaner := NewCleaner(zap.NewNop(), testMaster{}, []Target{{Dir: dir, MaxLifetime: lifetime}}, time.Hour)
	require.NoError(t, cleaner.Cleanup(t.Context(), now))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(dir, tt.file)
			if !tt.wantExists {
				assert.NoFileExists(t, path)
				return
			}

			info, err := os.Stat(path)
			require.NoError(t, err)

			wantModTime := tt.modTime
			if tt.open {
				wantModTime = now
			}
			assert.WithinDuration(t, wantModTime, info.ModTime(), time.Millisecond)
		})
	}
}

func TestCleaner_Cleanup_Dirs(t *testing.T) {
	now := time.Now()

	t.Run("missing directory is not an error", func(t *testing.T) {
		target := Target{Dir: filepath.Join(t.TempDir(), "missing"), MaxLifetime: time.Hour}
		cleaner := NewCleaner(zap.NewNop(), testMaster{}, []Target{target}, time.Hour)

		assert.NoError(t, cleaner.Cleanup(t.Context(), now))
	})

	t.Run("open session in a symlinked directory stays", func(t *testing.T) {
		realDir := t.TempDir()
		link := filepath.Join(t.TempDir(), "sessions")
		require.NoError(t, os.Symlink(realDir, link))

		path := filepath.Join(realDir, "sess_open")
		writeSessionFile(t, path, now.Add(-2*time.Hour))
		f, err := os.Open(path)
		require.NoError(t, err)
		t.Cleanup(func() { _ = f.Close() })

		cleaner := NewCleaner(zap.NewNop(), testMaster{}, []Target{{Dir: link, MaxLifetime: time.Hour}}, time.Hour)
		require.NoError(t, cleaner.Cleanup(t.Context(), now))

		assert.FileExists(t, path)
	})

	t.Run("canceled context stops the cleanup", func(t *testing.T) {
		dir := t.TempDir()
		writeSessionFile(t, filepath.Join(dir, "sess_expired"), now.Add(-2*time.Hour))

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		cleaner := NewCleaner(zap.NewNop(), testMaster{}, []Target{{Dir: dir, MaxLifetime: time.Hour}}, time.Hour)
		assert.ErrorIs(t, cleaner.Cleanup(ctx, now), context.Canceled)
		assert.FileExists(t, filepath.Join(dir, "sess_expired"))
	})
}

func TestCleaner_Run(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const (
			lifetime = time.Minute
			interval = time.Hour
		)
		dir := t.TempDir()
		cleaner := NewCleaner(zap.NewNop(), testMaster{}, []Target{{Dir: dir, MaxLifetime: lifetime}}, interval)

		atStart := filepath.Join(dir, "sess_at_start")
		writeSessionFile(t, atStart, time.Now().Add(-2*lifetime))

		ctx, cancel := context.WithCancel(t.Context())
		done := make(chan error, 1)
		go func() { done <- cleaner.Run(ctx) }()

		synctest.Wait()
		assert.NoFileExists(t, atStart, "Run cleans up at start")

		afterStart := filepath.Join(dir, "sess_after_start")
		writeSessionFile(t, afterStart, time.Now())

		time.Sleep(interval - time.Second)
		synctest.Wait()
		assert.FileExists(t, afterStart, "Run waits for the interval")

		time.Sleep(time.Second)
		synctest.Wait()
		assert.NoFileExists(t, afterStart, "Run cleans up after the interval")

		cancel()
		assert.NoError(t, <-done)
	})
}
