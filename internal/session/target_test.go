package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/code-tool/docker-fpm-wrapper/pkg/phpfpm"
)

func TestNewTargets(t *testing.T) {
	directives := map[string]string{
		"session.save_handler":   "files",
		"session.save_path":      "/var/lib/php/sessions",
		"session.gc_maxlifetime": "1440",
	}

	tests := []struct {
		name    string
		pools   []phpfpm.Pool
		want    []Target
		wantErr string
	}{
		{
			name:  "pool with php.ini directives",
			pools: []phpfpm.Pool{{Name: "www"}},
			want:  []Target{{Dir: "/var/lib/php/sessions", MaxLifetime: 1440 * time.Second}},
		},
		{
			name: "pools in one directory get the longest lifetime",
			pools: []phpfpm.Pool{
				{Name: "short", PhpValues: map[string]string{"session.gc_maxlifetime": "60"}},
				{Name: "long", PhpValues: map[string]string{"session.save_path": "2;/var/lib/php/sessions/", "session.gc_maxlifetime": "3600"}},
			},
			want: []Target{{Dir: "/var/lib/php/sessions", MaxLifetime: time.Hour}},
		},
		{
			name: "pools in different directories are sorted",
			pools: []phpfpm.Pool{
				{Name: "www"},
				{Name: "admin", PhpValues: map[string]string{"session.save_path": "/srv/admin-sessions"}},
			},
			want: []Target{
				{Dir: "/srv/admin-sessions", MaxLifetime: 1440 * time.Second},
				{Dir: "/var/lib/php/sessions", MaxLifetime: 1440 * time.Second},
			},
		},
		{
			name: "pools without session files are skipped",
			pools: []phpfpm.Pool{
				{Name: "redis", PhpValues: map[string]string{"session.save_handler": "redis"}},
				{Name: "tmp", PhpValues: map[string]string{"session.save_path": ""}},
			},
			want: []Target{},
		},
		{
			name:    "invalid lifetime",
			pools:   []phpfpm.Pool{{Name: "www", PhpValues: map[string]string{"session.gc_maxlifetime": "x"}}},
			wantErr: "pool www: parse session.gc_maxlifetime",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTargets(directives, tt.pools)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
