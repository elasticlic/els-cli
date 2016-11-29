package main_test

import (
	"bytes"
	"errors"

	cli "github.com/elasticlic/els-cli"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Password Test Suite", func() {

	var (
		setErr = errors.New("An Error")
	)

	Describe("StringPassworder", func() {
		var (
			sut *cli.StringPassworder
			p   = "password"
		)
		BeforeEach(func() {
			sut = cli.NewStringPassworder(p, setErr)
		})

		Describe("NewStringPassworder", func() {
			It("Creates the expected default struct", func() {
				Expect(*sut).To(BeEquivalentTo(cli.StringPassworder{
					Password: p,
					Error:    setErr,
				}))
			})
		})

		Describe("GetPassword", func() {
			It("Returns the simulated values", func() {
				var pw cli.Passworder = sut
				r, err := pw.GetPassword()
				Expect(r).To(Equal(p))
				Expect(err).To(Equal(setErr))
			})
		})
	})
	Describe("HiddenPassworder", func() {
		var (
			sut *cli.HiddenPassworder
			buf bytes.Buffer
		)
		BeforeEach(func() {
			sut = cli.NewHiddenPassworder(&buf)
		})

		It("Creates the expected default struct", func() {
			Expect(*sut).To(BeEquivalentTo(cli.HiddenPassworder{
				OutputStream: &buf,
			}))
		})

		// Note: Impossible to test terminal.ReadPassword, but the Passworder
		// interface was designed to make testing of the els-cli easier by
		// abstracting the collection of the password from the user, so in
		// practise we are leaving out the testing of
		// HiddenPassworder.GetPassword().
	})
})
