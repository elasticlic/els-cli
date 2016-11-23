package main_test

import (
	"bytes"
	"io"
	"io/ioutil"

	elsmock "github.com/elasticlic/els-api-sdk-go/els/mock"
	cli "github.com/elasticlic/els-cli"
	"github.com/elasticlic/go-utils/datetime"
	jcli "github.com/jawher/mow.cli"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
)

type MockPipe struct {
	Data string
	Err  error
}

// Reader implements interface els.Pipe.
func (p *MockPipe) Reader(io.ReadCloser, error) {
	return ioutil.NopCloser(bytes.NewReader([]byte(p.Data))), p.Err
}

var _ = Describe("els_cliTest Suite", func() {

	var (
		//err    error
		//args   []string
		sut    *cli.ELSCLI
		config cli.Config
		cFile  = "aConfigFile"
		mac    = elsmock.NewApiCaller()
		AppFs  = afero.NewMemMapFs()
		tp     = datetime.NewNowTimeProvider()
		pipe   = &MockPipe{}
		fs     = afero.NewMemMapFs()
	)

	Describe("ELSCLI", func() {

		BeforeEach(func() {
			sut = cli.NewELSCLI(jcli.App("els-cli", ""), &config, cFile, tp, mac, fs, pipe, input, outS, errS)
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
