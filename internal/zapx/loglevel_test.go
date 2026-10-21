package zapx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestLogLevel_UnmarshalFlag(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    zapcore.Level
		wantErr string
	}{
		{name: "name", value: "warn", want: zapcore.WarnLevel},
		{name: "upper case name", value: "ERROR", want: zapcore.ErrorLevel},
		{name: "negative number", value: "-1", want: zapcore.DebugLevel},
		{name: "number", value: "2", want: zapcore.ErrorLevel},
		{name: "unknown name", value: "verbose", wantErr: `unknown log level "verbose"`},
		{name: "number below debug", value: "-2", wantErr: "log level -2 is out of range [-1, 5]"},
		{name: "number above fatal", value: "6", wantErr: "log level 6 is out of range [-1, 5]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var level LogLevel
			err := level.UnmarshalFlag(tt.value)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, level.Level())
		})
	}
}
