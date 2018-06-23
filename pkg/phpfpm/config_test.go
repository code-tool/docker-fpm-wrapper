package phpfpm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	c, err := ParseConfig("testData/php-fpm.conf")
	assert.Nil(t, err)
	assert.Equal(t, "testData/php-fpm.d/*.conf", c.Include)

	assert.Equal(t, "www", c.Pools[0].Name)
	assert.Equal(t, "/run/php-fpm/php-fpm.sock", c.Pools[0].Listen)
	assert.Equal(t, "/status", c.Pools[0].StatusPath)
}
