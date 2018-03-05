package util

import (
	"bufio"
	"io"
)

func normalizeLine(line []byte) []byte {
	l := len(line)
	if l > 1 && line[l-2] == '\r' {
		line[l-2] = line[l-1]
		line = line[:l-1]
	}

	return line
}

func ReadLine(r *bufio.Reader) ([]byte, error) {
	skip := false

	for {
		line, err := r.ReadSlice('\n')

		switch err {
		case nil:
			if skip {
				return nil, nil
			}

			return normalizeLine(line), nil
		case bufio.ErrBufferFull:
			// TODO line is too long
			skip = true
		case io.EOF:
			// TODO badly-formed line
			return nil, nil
		default:
			return nil, err
		}
	}
}
