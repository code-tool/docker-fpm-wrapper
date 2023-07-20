package line

import (
	"bufio"
	"errors"
	"io"
)

func ReadOne(r *bufio.Reader, retBufOnEOF bool) ([]byte, error) {
	skip := false

	for {
		line, err := r.ReadSlice('\n')

		if errors.Is(err, io.EOF) {
			if retBufOnEOF {
				return line, nil
			}

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
