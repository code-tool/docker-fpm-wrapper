//go:build e2e

package e2e

import (
	"slices"
	"strings"
	"sync"

	"github.com/testcontainers/testcontainers-go"
)

// logCollector keeps the output of a container as lines.
type logCollector struct {
	mu      sync.Mutex
	lines   []string
	partial map[string]string // the incomplete last line of each stream
}

func newLogCollector() *logCollector {
	return &logCollector{partial: make(map[string]string)}
}

// Accept implements testcontainers.LogConsumer.
func (c *logCollector) Accept(l testcontainers.Log) {
	c.mu.Lock()
	defer c.mu.Unlock()

	data := c.partial[l.LogType] + string(l.Content)
	for {
		line, rest, found := strings.Cut(data, "\n")
		if !found {
			break
		}

		c.lines = append(c.lines, line)
		data = rest
	}

	c.partial[l.LogType] = data
}

// Lines returns a copy of the complete lines.
func (c *logCollector) Lines() []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return slices.Clone(c.lines)
}
