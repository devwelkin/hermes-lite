package headers

import (
	"bytes"
	"errors"
)

type Headers map[string]string

func NewHeaders() Headers {
	return map[string]string{}
}

func (h Headers) Parse(data []byte) (n int, done bool, err error) {
	idx := bytes.Index(data, []byte("\r\n"))

	if idx == -1 {
		return 0, false, nil
	}

	if idx == 0 {
		// the empty line
		return 2, true, nil
	}

	line := data[:idx]

	colonIdx := bytes.IndexByte(line, ':')
	if colonIdx == -1 {
		return 0, false, errors.New("invalid header: no colon found")
	}

	if colonIdx == 0 || line[colonIdx-1] == ' ' {
		return 0, false, errors.New("invalid header")
	}

	key := bytes.TrimSpace(line[:colonIdx])
	for _, b := range key {
		isLetter := (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
		isDigit := (b >= '0' && b <= '9')
		// bytes.IndexByte is faster than looping a string
		isSpecial := bytes.IndexByte([]byte("!#$%&'*+-.^_`|~"), b) != -1

		if !isLetter && !isDigit && !isSpecial {
			return 0, false, errors.New("invalid header key: invalid character")
		}
	}

	// NEW: always lowercase the key
	key = bytes.ToLower(key)
	value := bytes.TrimSpace(line[colonIdx+1:])

	_, ok := h[string(key)]
	if ok {
		h[string(key)] += ", " + string(value)
		return idx + 2, false, nil
	}

	h[string(key)] = string(value)

	return idx + 2, false, nil
}
