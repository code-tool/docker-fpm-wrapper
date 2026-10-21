package phpfpm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	dir := "/tmp/fpm-test"
	assert.NoError(t, os.Mkdir(dir, 0777))
	assert.NoError(t, copy.Copy("testdata", dir))

	defer func() { _ = os.RemoveAll(dir) }()

	c, err := ParseConfig(dir + "/php-fpm.conf")
	require.NoError(t, err)
	assert.Equal(t, dir+"/php-fpm.d/*.conf", c.Include)

	assert.Equal(t, []Pool{
		{
			Name:                     "nostatus",
			Listen:                   "/run/php-fpm/nostatus.sock",
			StatusListen:             "/run/php-fpm/nostatus-status.sock",
			SlowlogPath:              "/var/log/nostatus/nostatus.slow.log",
			RequestSlowlogTimeout:    5,
			RequestSlowlogTraceDepth: 64,
			PhpValues:                map[string]string{"session.save_path": "/var/lib/php/nostatus-sessions"},
		},
		{
			Name:                     "www",
			Listen:                   "/run/php-fpm/php-fpm.sock",
			StatusPath:               "/status",
			SlowlogPath:              "log/www.log.slow",
			RequestSlowlogTimeout:    1,
			RequestSlowlogTraceDepth: 64,
			PhpValues: map[string]string{
				"session.save_path":      "2;/var/lib/php/sessions",
				"session.gc_maxlifetime": "3600",
				"log_errors":             "on",
			},
		},
	}, c.Pools)
}

func TestParseConfig_Errors(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr string
	}{
		{
			name:    "no global section",
			config:  "[www]\nlisten = 9000\n",
			wantErr: `section "global" does not exist`,
		},
		{
			name:    "pool without listen",
			config:  "[global]\nerror_log = /proc/self/fd/2\n\n[www]\nlisten = 9000\n\n[broken]\npm = static\n",
			wantErr: "pool broken: ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fPath := filepath.Join(t.TempDir(), "php-fpm.conf")
			require.NoError(t, os.WriteFile(fPath, []byte(tt.config), 0o600))

			_, err := ParseConfig(fPath)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}

	t.Run("missing file", func(t *testing.T) {
		_, err := ParseConfig(filepath.Join(t.TempDir(), "php-fpm.conf"))
		assert.ErrorContains(t, err, "no such file or directory")
	})
}
