package main_test

import (
	"bytes"
	"io"
	"io/ioutil"
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
		config      cli.Config
		cFile       = "aConfigFile"
		jFile       = "t.json"
		pw          = "password"
		pwr         = cli.NewStringPassworder(pw, nil)
		ID          = els.AccessKeyID("anID")
		SAC         = els.SecretAccessKey("aSAC")
		email       = "email@example.com"
		expiry      = time.Now().Add(time.Hour)
		reqJ        = `{"send":"aValue"}`
		repJ        = `{"rec":"aValue"}`
		prof        *cli.Profile
		vendorID        = "aVendor"
		rulesetID       = "aRuleset"
		maxAPITries int = 1
	)

	Describe("ELSCLI", func() {

		var (
			sut  *cli.ELSCLI
			fr   *jcli.Cli
			args []string
			ac   *em.APICaller
			outS bytes.Buffer
			errS bytes.Buffer
			pipe *MockPipe
			fs   afero.Fs
			tp   *datetime.NowTimeProvider

			// checkSentContent checks if the els-cli passed on the expected content
			// in the body of the request to the ELS.
			checkSentContent = func() {
				sentJ, err := ioutil.ReadAll(ac.GetCall(0).ACArgs.Req.Body)
				Expect(err).To(BeNil())
				Expect(sentJ).To(MatchJSON(reqJ))
			}

			// checkOutputContent checks if the els-cli emitted the JSON response
			// body received from the ELS.
			checkOutputContent = func() {
				Expect(errS.String()).To(BeZero())
				Expect(outS.String()).To(MatchJSON(repJ))
			}

			checkRequest = func(httpMethod string, URL string) {
				Expect(httpMethod).To(Equal(ac.GetCall(0).ACArgs.Req.Method))
				Expect(URL).To(Equal(ac.GetCall(0).ACArgs.Req.URL.Path))
			}

			// initAPIResponse sets a simple expectation on an APICaller method
			// being invoked and a response returned.
			initResponse = func(callMethod string, statusCode int) {
				ac.AddExpectedCall(callMethod, em.APICall{
					ACRep: em.ACRep{Rep: em.HTTPResponse(statusCode, repJ)},
				})
			}
		)

		BeforeEach(func() {
			args = []string{"els-cli"}
			ac = em.NewAPICaller()
			pipe = &MockPipe{}
			fs = afero.NewMemMapFs()
			tp = datetime.NewNowTimeProvider()
			fr = jcli.App("els-cli", "")

			outS = bytes.Buffer{}
			errS = bytes.Buffer{}
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

			sut = cli.NewELSCLI(fr, &config, cFile, tp, fs, ac, pipe, pwr, &outS, &errS)
		})

		JustBeforeEach(func() {
			sut.Run(args)
		})

		Describe("General Response Processing", func() {
			// These tests are the only place we'll test the pipe input and
			// the different output types
			BeforeEach(func() {
				args = append(args, "vendors", vendorID, "put")
			})
			Context("JSON is piped to the command-line", func() {
				BeforeEach(func() {
					pipe.Data = reqJ

					initResponse("Do", 200)
				})
				It("Receives a result from the API", func() {
					checkSentContent()
					checkOutputContent()
				})
			})
			Context("A file containing JSON is specified on the commandline", func() {
				BeforeEach(func() {
					args = append(args, jFile)

					// Add a file to our FS simulation which the els-cli will
					// access:
					afero.WriteFile(fs, jFile, []byte(reqJ), 0644)

					initResponse("Do", 200)
				})
				It("Sends the correct content and outputs the  API response body", func() {
					checkSentContent()
					checkOutputContent()
				})
			})
		})

		Describe("vendor", func() {
			BeforeEach(func() {
				args = append(args, "vendors", vendorID)
			})

			Describe("put", func() {
				BeforeEach(func() {
					args = append(args, "put")
				})
				Context("JSON is piped to the command-line", func() {
					BeforeEach(func() {
						pipe.Data = reqJ
						initResponse("Do", 200)
					})
					It("Receives a result from the API", func() {
						checkRequest("PUT", "/vendors/"+vendorID)
						checkSentContent()
						checkOutputContent()
					})
				})
			})
			Describe("get", func() {
				BeforeEach(func() {
					args = append(args, "get")
					initResponse("Do", 200)
				})

				It("Receives a result from the API", func() {
					checkRequest("GET", "/vendors/"+vendorID)
					checkOutputContent()
				})
			})
			Describe("rulesets", func() {
				BeforeEach(func() {
					args = append(args, "rulesets")
				})

				Describe("put", func() {
					BeforeEach(func() {
						args = append(args, rulesetID, "put")
						initResponse("Do", 200)
					})
					It("Receives a result from the API", func() {
						checkRequest("PUT", "/vendors/"+vendorID+"/paygRuleSets/"+rulesetID)
						checkOutputContent()
					})
				})
				Describe("activate", func() {
					BeforeEach(func() {
						args = append(args, rulesetID, "activate")
						initResponse("Do", 204)
					})
					It("Receives a result from the API", func() {
						checkRequest("PATCH", "/vendors/"+vendorID+"/paygRuleSets/"+rulesetID+"/activate")
						checkOutputContent()
					})
				})
				XDescribe("get all rulesets", func() {
					BeforeEach(func() {
						args = append(args, "get")
						initResponse("Do", 200)
					})
					It("Receives a result from the API", func() {
						checkRequest("PUT", "/vendors/"+vendorID+"/paygRuleSets")
						checkOutputContent()
					})
				})
			})
		})
	})
})
