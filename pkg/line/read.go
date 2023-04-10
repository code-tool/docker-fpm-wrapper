package line

import (
	"bufio"
	"io"
)

func ReadOne(r *bufio.Reader) ([]byte, error) {
	skip := false

	for {
		line, err := r.ReadSlice('\n')

		switch err {
		case nil:
			if skip {
				skip = false
				continue
			}

			return line, nil
		case bufio.ErrBufferFull:
			// line is too long
			skip = true
		case io.EOF:
			// badly-formed line
			return nil, io.EOF
		default:
			return nil, err
		}
	}
}
