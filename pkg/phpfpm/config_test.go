package phpfpm

import (
	"os"
	"testing"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	dir := "/tmp/fpm-test"
	assert.NoError(t, os.Mkdir(dir, 0777))
	assert.NoError(t, copy.Copy("testdata", dir))

	defer func() { _ = os.RemoveAll(dir) }()

	c, err := ParseConfig(dir + "/php-fpm.conf")
	assert.Nil(t, err)
	assert.Equal(t, dir+"/php-fpm.d/*.conf", c.Include)

	assert.Equal(t, "www", c.Pools[0].Name)
	assert.Equal(t, 1, c.Pools[0].RequestSlowlogTimeout)
	assert.Equal(t, "/run/php-fpm/php-fpm.sock", c.Pools[0].Listen)
	assert.Equal(t, "/status", c.Pools[0].StatusPath)
	assert.Equal(t, "log/www.log.slow", c.Pools[0].SlowlogPath)
}
