package phpfpm

import (
	"testing"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	"os"
)

func TestParse(t *testing.T) {
	dir := "/tmp/fpm-test"
	assert.NoError(t, os.Mkdir(dir, 0777))
	copy.Copy("testdata", dir)

	defer os.RemoveAll(dir)

	c, err := ParseConfig(dir + "/php-fpm.conf")
	assert.Nil(t, err)
	assert.Equal(t, dir + "/php-fpm.d/*.conf", c.Include)

	assert.Equal(t, "www", c.Pools[0].Name)
	assert.Equal(t, "/run/php-fpm/php-fpm.sock", c.Pools[0].Listen)
	assert.Equal(t, "/status", c.Pools[0].StatusPath)
}
