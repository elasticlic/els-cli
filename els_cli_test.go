package main_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/elasticlic/els-api-sdk-go/els"
	em "github.com/elasticlic/els-api-sdk-go/els/mock"
	cli "github.com/elasticlic/els-cli"
	"github.com/elasticlic/go-utils/datetime"
	jcli "github.com/jawher/mow.cli"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
)

// MockPipe is used to simulate piped input to the command via the commandline.
type MockPipe struct {
	Data string
	Err  error
}

// Reader implements interface main.Pipe and presents our test data in the
// MockPipe.
func (p *MockPipe) Reader() (io.ReadCloser, error) {
	return ioutil.NopCloser(bytes.NewReader([]byte(p.Data))), p.Err
}

var _ = Describe("els_cliTest Suite", func() {

	var (
		//err    error
		args        []string
		sut         *cli.ELSCLI
		fr          = jcli.App("els-cli", "")
		config      cli.Config
		cFile       = "aConfigFile"
		ac          = em.NewAPICaller()
		tp          = datetime.NewNowTimeProvider()
		pipe        = &MockPipe{}
		fs          = afero.NewMemMapFs()
		pw          = "password"
		inS         = strings.NewReader(pw)
		outS        bytes.Buffer
		errS        bytes.Buffer
		ID          = els.AccessKeyID("anID")
		SAC         = els.SecretAccessKey("aSAC")
		email       = "email@example.com"
		expiry      = time.Now().Add(time.Hour)
		validJ      = `{"aField":"aValue"}`
		prof        *cli.Profile
		vendorId        = "aVendor"
		maxAPITries int = 1
	)

	Describe("ELSCLI", func() {

		BeforeEach(func() {
			args = append(args, "els-cli")
			config.Profiles = make(map[string]*cli.Profile)
			config.Profiles["default"] = &cli.Profile{
				AccessKey: els.AccessKey{
					ID:              ID,
					SecretAccessKey: SAC,
					Email:           email,
					ExpiryDate:      expiry,
				},
				MaxAPITries: maxAPITries,
				Output:      cli.OutputBodyOnly,
			}
			prof = config.Profiles["default"]

			sut = cli.NewELSCLI(fr, &config, cFile, tp, fs, ac, pipe, inS, &outS, &errS)
		})

		JustBeforeEach(func() {
			sut.Run(args)
		})

		Describe("General Response Processing", func() {
			// These tests are the only place we'll test the pipe input and
			// the different output types
			BeforeEach(func() {
				args = append(args, "vendor", vendorId, "put")
			})
			Context("JSON is piped to the command-line", func() {
				BeforeEach(func() {
					pipe.Data = validJ

					ac.AddExpectedCall("Do", em.APICall{
						ACRep: em.ACRep{Rep: em.HTTPResponse(200, validJ)},
					})

				})
				It("Receives a result from the API", func() {
					Expect(errS.String()).To(BeZero())
					Expect(outS.String()).To(MatchJSON(validJ))
				})
			})
		})

		XDescribe("vendor", func() {
			BeforeEach(func() {
				args = append(args, "vendor", vendorId)
			})

			Describe("put", func() {
				BeforeEach(func() {
					args = append(args, "put")
				})
				// This is the only place where we'll test the pipe argument
				Context("JSON is piped to the command-line", func() {
					BeforeEach(func() {
						pipe.Data = validJ

						ac.AddExpectedCall("Do", em.APICall{
							ACRep: em.ACRep{Rep: em.HTTPResponse(200, validJ)},
						})

					})
					It("Receives a result from the API", func() {
						Expect(errS.String()).To(BeZero())
						Expect(outS.String()).To(MatchJSON(validJ))
					})
				})
			})
		})
	})
})
