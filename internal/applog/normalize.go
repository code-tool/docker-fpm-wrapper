package applog

func normalizeLine(line []byte) []byte {
	ll := len(line)
	if ll > 0 && line[ll-1] != '\n' {
		ll += 1
		line = append(line, '\n')
	}

	if ll > 1 && line[ll-2] == '\r' {
		line[ll-2] = line[ll-1]
		line = line[:ll-1]
	}

	return line
}
