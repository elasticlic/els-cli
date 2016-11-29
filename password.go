package main

import (
	"fmt"
	"io"

	"golang.org/x/crypto/ssh/terminal"
)

// Passworder is the interface defining a method to obtain a password entered
// by the user.
type Passworder interface {
	GetPassword() (string, error)
}

// StringPassworder is used to satisfy interface Passworder using a predefined
// string and error, which is used in testing.
type StringPassworder struct {
	Password string
	Error    error
}

func NewStringPassworder(p string, err error) *StringPassworder {
	return &StringPassworder{
		Password: p,
		Error:    err,
	}
}

// GetPassword implements interface Passworder
func (p *StringPassworder) GetPassword() (string, error) {
	return p.Password, p.Error
}

// HiddenPassworder is used to provide a password obtained silently from the
// user via the command-line.
type HiddenPassworder struct {
	// OutputStream is the stream to which prompts to the user are written
	OutputStream io.Writer
}

// NewHiddenPassworder returns a HiddenPassworder which will write prompts to
// the given writer.
func NewHiddenPassworder(outS io.Writer) *HiddenPassworder {
	return &HiddenPassworder{
		OutputStream: outS,
	}
}

// GetPassword implements interface Passworder and asks for a password from the
// user on the command-line, hiding their input. The returned string does not
// have a newline character at the end.
func (p *HiddenPassworder) GetPassword() (string, error) {

	fmt.Fprintln(p.OutputStream, "Enter password: ")

	b, err := terminal.ReadPassword(0)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
