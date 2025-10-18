package main

import (
	"bytes"
	"io"
)

func GetLinesChannel(f io.ReadCloser) <-chan string {
	strChan := make(chan string)

	// Create a bucket and accumulator
	data := make([]byte, 8)
	var linebuf bytes.Buffer

	go func() {
		defer close(strChan)
		defer f.Close()

		for {
			// Read to bucket
			n, err := f.Read(data)

			if n > 0 {

				// Take the chunk you read
				chunk := data[:n]

				// start searching for '\n' inside the chunk
				for {
					// find index of '\n'
					i := bytes.IndexByte(chunk, '\n')

					if i == -1 {
						// '\n' not found
						linebuf.Write(chunk)
						break
					}
					// add the part up to '\n' ('chunk[:i]') to the accumulator
					linebuf.Write(chunk[:i])

					//  send line to channel
					strChan <- linebuf.String()

					//  reset accumulator
					linebuf.Reset()

					chunk = chunk[i+1:]
				}
			}

			if err == io.EOF {
				if linebuf.Len() > 0 {
					strChan <- linebuf.String()
				}
				break
			}

			if err != nil {
				break
			}
		}
	}()

	return strChan
}
