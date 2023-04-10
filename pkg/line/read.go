package line

import (
	"bufio"
	"errors"
	"io"
)

func ReadOne(r *bufio.Reader) ([]byte, error) {
	skip := false

	for {
		line, err := r.ReadSlice('\n')

		if errors.Is(err, io.EOF) {
			return nil, err
		}

		if errors.Is(err, bufio.ErrBufferFull) {
			// line is too long
			skip = true
			continue
		}

		if err != nil {
			return nil, err
		}

		if skip {
			skip = false
			continue
		}

		return line, nil
	}
}
