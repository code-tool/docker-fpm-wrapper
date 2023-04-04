package phpfpm

import (
	"errors"
	"io"
	"os"
	"sync"
	"testing"
)

func TestSlowlogParser(t *testing.T) {
	f, err := os.Open("testdata/slowlog.log")
	if err != nil {
		t.Error(err)
		t.Fail()
		return
	}

	slp := &SlowlogParser{}
	out := make(chan SlowlogEntry)

	pipeReader, pipeWriter := io.Pipe()

	go func() {
		defer close(out)
		if err := slp.Parse(pipeReader, out); err != nil && !errors.Is(err, io.EOF) {
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
			pipeWriter.CloseWithError(err)
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
	if len(entries) != 2 {
		t.Fail()
	}
}
