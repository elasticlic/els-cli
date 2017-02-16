package main

import (
	"io"
	"os"
)

// Pipe is the interface which defines methods operating on data piped to
// the command via the command-line.
type Pipe interface {
	Reader() (io.ReadCloser, error)
}

// CLIPipe is used to read data piped to the els-cli from the commmand-line.
type CLIPipe struct{}

func NewCLIPipe() *CLIPipe {
	return &CLIPipe{}
}

// Reader implements interface Pipe and returns a Reader which can be used to
// read the data passed to the els-cli via a command-line pipe.
func (p *CLIPipe) Reader() (io.ReadCloser, error) {
	info, err := os.Stdin.Stat()

	if err != nil {
		return nil, err
	}

	if (info.Mode() & os.ModeCharDevice) != 0 {
		// no data from a pipe - ignore
		return nil, ErrNoContent
	}

	return os.Stdin, nil
}
