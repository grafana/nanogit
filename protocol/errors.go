package protocol

import (
	"errors"
	"io"
)

type strError string

func (e strError) Error() string {
	return string(e)
}

// eofIsUnexpected checks if the error is an io.EOF.
// If it is, we return io.ErrUnexpectedEOF.
// If not, we return the input error verbatim.
func eofIsUnexpected(err error) error {
	if errors.Is(err, io.EOF) {
		return io.ErrUnexpectedEOF
	} else {
		return err
	}
}
