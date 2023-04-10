package phpfpm

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlowlogParser(t *testing.T) {
	f, err := os.Open("testdata/slowlog.log")
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}
	defer f.Close()

	slp := NewSlowlogParser(0)
	out := make(chan SlowlogEntry)

	pipeReader, pipeWriter := io.Pipe()

	go func() {
		defer close(out)
		if err := slp.Parse(context.TODO(), pipeReader, out); err != nil && !errors.Is(err, io.EOF) {
			t.Fail()
		}
	}()

	var wg sync.WaitGroup
	var entries []SlowlogEntry

	wg.Add(1)
	go func() {
		defer wg.Done()
		for e := range out {
			entries = append(entries, e)
		}
	}()

	buf := make([]byte, 10)
	for {
		n, err := f.Read(buf)
		if err != nil {
			assert.NoError(t, pipeWriter.CloseWithError(err))
			break
		}

		if _, err := pipeWriter.Write(buf[0:n]); err != nil {
			t.Error(err)
			t.Fail()
			break
		}
	}
	//

	wg.Wait()
	if !assert.Equal(t, 2, len(entries)) {
		return
	}

	_, err = f.Seek(0, 0)
	assert.NoError(t, err)

	allContent, err := io.ReadAll(f)
	assert.NoError(t, err)

	assert.True(t, bytes.Contains(allContent, []byte(entries[0].String())))
}
