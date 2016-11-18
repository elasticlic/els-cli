package main_test

import (
	cli "github.com/elasticlic/els-cli"
	jcli "github.com/jawher/mow.cli"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("els_cliTest Suite", func() {

	var (
		//err    error
		//args   []string
		sut    *cli.ELSCLI
		config cli.Config
		cFile  = "aConfigFile"
	)

	Describe("ELSCLI", func() {

		BeforeEach(func() {
			sut = cli.NewELSCLI(jcli.App("els-cli", ""), &config, cFile)
		})

		Describe("Run", func() {
			Context("", func() {
				It("Returns an ELSCLI", func() {
					Expect(true).To(BeTrue())
				})
			})
		})
	})
})
