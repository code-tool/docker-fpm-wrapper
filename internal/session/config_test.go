package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	directives := map[string]string{
		"session.save_handler":   "files",
		"session.save_path":      "/var/lib/php/sessions",
		"session.gc_maxlifetime": "1440",
	}

	tests := []struct {
		name       string
		directives map[string]string
		poolValues map[string]string
		want       Config
		wantErr    string
	}{
		{
			name:       "directives of php.ini",
			directives: directives,
			want:       Config{SaveHandler: "files", SavePath: "/var/lib/php/sessions", GcMaxLifetime: 1440 * time.Second},
		},
		{
			name:       "pool values take precedence",
			directives: directives,
			poolValues: map[string]string{"session.save_path": "/srv/sessions", "session.gc_maxlifetime": "60"},
			want:       Config{SaveHandler: "files", SavePath: "/srv/sessions", GcMaxLifetime: time.Minute},
		},
		{
			name:       "missing lifetime has the PHP default",
			directives: map[string]string{"session.save_handler": "files"},
			want:       Config{SaveHandler: "files", GcMaxLifetime: defaultGcMaxLifetime},
		},
		{
			name:       "invalid lifetime",
			directives: directives,
			poolValues: map[string]string{"session.gc_maxlifetime": "1h"},
			wantErr:    "parse session.gc_maxlifetime",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewConfig(tt.directives, tt.poolValues)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfig_Dir(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		wantDir  string
		hasFiles bool
	}{
		{name: "path", config: Config{SaveHandler: "files", SavePath: "/var/lib/php/sessions"}, wantDir: "/var/lib/php/sessions", hasFiles: true},
		{name: "levels and path", config: Config{SaveHandler: "files", SavePath: "2;/var/lib/php/sessions"}, wantDir: "/var/lib/php/sessions", hasFiles: true},
		{name: "levels, mode, and path", config: Config{SaveHandler: "files", SavePath: "2;600;/var/lib/php/sessions"}, wantDir: "/var/lib/php/sessions", hasFiles: true},
		{name: "empty path", config: Config{SaveHandler: "files"}, wantDir: "", hasFiles: false},
		{name: "other handler", config: Config{SaveHandler: "redis", SavePath: "tcp://redis:6379"}, wantDir: "tcp://redis:6379", hasFiles: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantDir, tt.config.Dir())
			assert.Equal(t, tt.hasFiles, tt.config.HasFiles())
		})
	}
}
