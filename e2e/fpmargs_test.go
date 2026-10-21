//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFpmArgs examines the arguments after "--": the wrapper passes them to php-fpm.
func TestFpmArgs(t *testing.T) {
	t.Parallel()

	w := startWrapper(t, "--log-level", "-1", "--", "-d", "memory_limit=77M")

	assert.Equal(t, "77M", w.mustRequest(t, wwwPort, "ini.php", "name=memory_limit"))
}
