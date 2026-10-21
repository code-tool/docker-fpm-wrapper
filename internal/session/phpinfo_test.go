package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePhpinfo(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{
			name: "directives",
			input: "session\n\n" +
				"Directive => Local Value => Master Value\n" +
				"session.gc_maxlifetime => 60 => 1440\n" +
				"session.save_handler => files => files\n",
			want: map[string]string{
				"Directive":              "Local Value",
				"session.gc_maxlifetime": "60",
				"session.save_handler":   "files",
			},
		},
		{
			name:  "no value is empty",
			input: "session.save_path => no value => no value\n",
			want:  map[string]string{"session.save_path": ""},
		},
		{
			name:  "rows with two columns are skipped",
			input: "Server API => FPM/FastCGI\nRegistered save handlers => files user\n",
			want:  map[string]string{},
		},
		{
			name:  "empty output",
			input: "",
			want:  map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePhpinfo(strings.NewReader(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestReadFpmDirectives(t *testing.T) {
	tests := []struct {
		name    string
		script  string
		args    []string
		want    map[string]string
		wantErr string
	}{
		{
			name:   "arguments go before -i",
			script: "#!/bin/sh\necho \"args => $* => master\"\n",
			args:   []string{"-d", "session.gc_maxlifetime=60"},
			want:   map[string]string{"args": "-d session.gc_maxlifetime=60 -i"},
		},
		{
			name:    "php-fpm fails",
			script:  "#!/bin/sh\necho 'unknown option' >&2\nexit 64\n",
			wantErr: "exit status 64: unknown option",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fpmPath := filepath.Join(t.TempDir(), "php-fpm")
			require.NoError(t, os.WriteFile(fpmPath, []byte(tt.script), 0o755))

			got, err := ReadFpmDirectives(t.Context(), fpmPath, tt.args)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("php-fpm does not exist", func(t *testing.T) {
		_, err := ReadFpmDirectives(t.Context(), filepath.Join(t.TempDir(), "php-fpm"), nil)
		assert.ErrorContains(t, err, "no such file or directory")
	})
}
