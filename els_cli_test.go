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
		fatalErr        error
		config          cli.Config
		cFile           = "aConfigFile"
		jFile           = "t.json"
		pw              = "password"
		expiryDays      = 30
		pwr             = cli.NewStringPassworder(pw, nil)
		ID              = els.AccessKeyID("anID")
		SAC             = els.SecretAccessKey("aSAC")
		email           = "email@example.com"
		expiry          = time.Now().Add(time.Hour)
		reqJ            = `{"send":"aValue"}`
		repJ            = `{"rec":"aValue"}`
		prof            *cli.Profile
		vendorID            = "aVendor"
		cloudProviderID     = "aCloudProvider"
		rulesetID           = "aRuleset"
		URL                 = "/a/path/and?querystring"
		maxAPITries     int = 1
		accessKey           = els.AccessKey{
			ID:              ID,
			SecretAccessKey: SAC,
			Email:           email,
			ExpiryDate:      expiry,
		}
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
			checkSentContent = func(json string) {
				sentJ, err := ioutil.ReadAll(ac.GetCall(0).ACArgs.Req.Body)
				Expect(err).To(BeNil())
				Expect(sentJ).To(MatchJSON(json))
			}

			// checkOutputContent checks if the els-cli emitted the JSON response
			// body received from the ELS.
			checkOutputContent = func(json string) {
				Expect(errS.String()).To(BeZero())
				if json != "" {
					Expect(outS.String()).To(MatchJSON(json))
				}
			}

			checkRequest = func(httpMethod string, URL string) {
				r := ac.GetCall(0).ACArgs.Req
				Expect(httpMethod).To(Equal(r.Method))

				u := r.URL
				if u.RawQuery != "" {
					Expect(URL).To(Equal(u.Path + "?" + u.RawQuery))
				} else {
					Expect(URL).To(Equal(u.Path))
				}
			}

			// initAPIResponse sets a simple expectation on an APICaller method
			// being invoked and a response returned.
			initResponse = func(callMethod string, statusCode int, repJson string) {
				ac.AddExpectedCall(callMethod, em.APICall{
					ACRep: em.ACRep{Rep: em.HTTPResponse(statusCode, repJson)},
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
				AccessKey:   accessKey,
				MaxAPITries: maxAPITries,
				Output:      cli.OutputBodyOnly,
			}
			prof = config.Profiles["default"]

			sut = cli.NewELSCLI(fr, &config, cFile, tp, fs, ac, pipe, pwr, &outS, &errS)
		})

		JustBeforeEach(func() {
			fatalErr = sut.Run(args)
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

					initResponse("Do", 200, repJ)
				})
				It("Receives a result from the API", func() {
					checkSentContent(reqJ)
					checkOutputContent(repJ)
				})
			})
			Context("A file containing JSON is specified on the commandline", func() {
				BeforeEach(func() {
					args = append(args, jFile)

					// Add a file to our FS simulation which the els-cli will
					// access:
					afero.WriteFile(fs, jFile, []byte(reqJ), 0644)

				})
				Context("200 is returned", func() {
					BeforeEach(func() {
						initResponse("Do", 200, repJ)
					})
					It("Sends the correct content and outputs the  API response body", func() {
						checkSentContent(reqJ)
						checkOutputContent(repJ)
					})
				})

				Context("401 is retured", func() {
					BeforeEach(func() {
						config.Profiles["default"].Output = cli.OutputWhole
						initResponse("Do", 401, "")
					})
					It("Reports the error", func() {
						checkSentContent(reqJ)
						Expect(outS.String()).Should(HavePrefix("401"))
					})
				})
			})
		})

		Describe("user", func() {
			BeforeEach(func() {
				args = append(args, "users", email)
			})

			Describe("accessKey", func() {
				BeforeEach(func() {
					args = append(args, "accessKeys")
				})
				Describe("create", func() {
					BeforeEach(func() {
						args = append(args, "create")
					})

					Context("The correct password is entered", func() {
						BeforeEach(func() {
							ac.AddExpectedCall("CreateAccessKey", em.APICall{
								ACRep: em.ACRep{
									StatusCode: 201,
									AccessKey:  &accessKey,
								},
							})
						})
						It("Returns the expected access key", func() {
							args := ac.GetCall(0).ACArgs
							Expect(args.EmailAddress).To(Equal(email))
							Expect(args.Password).To(Equal(pw))
							Expect(args.ExpiryDays).To(BeEquivalentTo(expiryDays))

							Expect(outS.String()).Should(HavePrefix("Access Key Created"))
							Expect(outS.String()).Should(ContainSubstring(accessKey.Email))
							Expect(outS.String()).Should(ContainSubstring(string(accessKey.SecretAccessKey)))
							Expect(outS.String()).Should(ContainSubstring(string(accessKey.ID)))
						})
					})
					Context("An invalid password is entered", func() {
						BeforeEach(func() {
							ac.AddExpectedCall("CreateAccessKey", em.APICall{
								ACRep: em.ACRep{
									StatusCode: 401,
								},
							})
						})
						It("Returns the failure statuscode", func() {
							args := ac.GetCall(0).ACArgs
							Expect(args.EmailAddress).To(Equal(email))
							Expect(args.Password).To(Equal(pw))
							Expect(args.ExpiryDays).To(BeEquivalentTo(expiryDays))
							Expect(outS.String()).Should(HavePrefix("The email address or password are incorrect"))
						})
					})
				})
				Describe("delete", func() {
					BeforeEach(func() {
						args = append(args, "delete", string(ID))
					})
					Context("The request succeeds", func() {
						BeforeEach(func() {
							config.Profiles["default"].Output = cli.OutputWhole
							initResponse("Do", 204, "")
						})
						It("returns the success code", func() {
							checkRequest("DELETE", "/users/"+email+"/accessKeys/"+string(ID))
							Expect(outS.String()).Should(HavePrefix("204"))
						})
					})
					Context("The user isn't permitted to make the call", func() {
						BeforeEach(func() {
							config.Profiles["default"].Output = cli.OutputWhole
							initResponse("Do", 401, "")
						})
						It("returns the failure code", func() {
							checkRequest("DELETE", "/users/"+email+"/accessKeys/"+string(ID))
							Expect(outS.String()).Should(HavePrefix("401"))
						})
					})
				})
				Describe("list", func() {
					BeforeEach(func() {
						args = append(args, "list")
					})
					Context("The request succeeds", func() {
						BeforeEach(func() {
							initResponse("Do", 200, repJ)
						})
						It("returns the success code", func() {
							checkRequest("GET", "/users/"+email+"/accessKeys")
							Expect(outS.String()).Should(MatchJSON(repJ))
						})
					})
					Context("The user isn't permitted to make the call", func() {
						BeforeEach(func() {
							config.Profiles["default"].Output = cli.OutputWhole
							initResponse("Do", 401, "")
						})
						It("returns the failure code", func() {
							checkRequest("GET", "/users/"+email+"/accessKeys")
							Expect(outS.String()).Should(HavePrefix("401"))
						})
					})
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
						initResponse("Do", 200, repJ)
					})
					It("Receives a result from the API", func() {
						checkRequest("PUT", "/vendors/"+vendorID)
						checkSentContent(reqJ)
						checkOutputContent(repJ)
					})
				})
			})
			Describe("get", func() {
				BeforeEach(func() {
					args = append(args, "get")
					initResponse("Do", 200, repJ)
				})

				It("Receives a result from the API", func() {
					checkRequest("GET", "/vendors/"+vendorID)
					checkOutputContent(repJ)
				})
			})
			Describe("list-rulesets", func() {
				BeforeEach(func() {
					args = append(args, "list-rulesets")
					initResponse("Do", 200, repJ)
				})
				It("Receives a result from the API", func() {
					checkRequest("GET", "/vendors/"+vendorID+"/paygRuleSets")
					checkOutputContent(repJ)
				})
			})
			Describe("rulesets", func() {
				BeforeEach(func() {
					args = append(args, "rulesets")
				})

				Describe("put", func() {
					BeforeEach(func() {
						args = append(args, rulesetID, "put")
						initResponse("Do", 200, repJ)
					})
					It("Receives a result from the API", func() {
						checkRequest("PUT", "/vendors/"+vendorID+"/paygRuleSets/"+rulesetID)
						checkOutputContent(repJ)
					})
				})
				Describe("get", func() {
					BeforeEach(func() {
						args = append(args, rulesetID, "get")
						initResponse("Do", 200, repJ)
					})
					It("Receives a result from the API", func() {
						checkRequest("GET", "/vendors/"+vendorID+"/paygRuleSets/"+rulesetID)
						checkOutputContent(repJ)
					})
				})
				Describe("activate", func() {
					BeforeEach(func() {
						args = append(args, rulesetID, "activate")
						initResponse("Do", 204, "")
					})
					It("Receives a result from the API", func() {
						checkRequest("PATCH", "/vendors/"+vendorID+"/paygRuleSets/"+rulesetID+"/activate")
						checkOutputContent("")
					})
				})
			})
		})

		Describe("cloud-provider", func() {
			BeforeEach(func() {
				args = append(args, "cloud-providers", cloudProviderID)
			})

			Describe("put", func() {
				BeforeEach(func() {
					args = append(args, "put")
				})
				Context("JSON is piped to the command-line", func() {
					BeforeEach(func() {
						pipe.Data = reqJ
						initResponse("Do", 200, repJ)
					})
					It("Receives a result from the API", func() {
						checkRequest("PUT", "/partners/"+cloudProviderID)
						checkSentContent(reqJ)
						checkOutputContent(repJ)
					})
				})
			})
			Describe("get", func() {
				BeforeEach(func() {
					args = append(args, "get")
					initResponse("Do", 200, repJ)
				})

				It("Receives a result from the API", func() {
					checkRequest("GET", "/partners/"+cloudProviderID)
					checkOutputContent(repJ)
				})
			})
		})
		Describe("do", func() {
			BeforeEach(func() {
				args = append(args, "do")
			})
			// Note in the tests below, we submit the URL WITHOUT the leading
			// slash... We expect that to be added automatically...
			Describe("GET", func() {
				BeforeEach(func() {
					args = append(args, "GET", URL[1:])
					initResponse("Do", 200, repJ)
				})

				It("Receives a result from the API", func() {
					checkRequest("GET", URL)
					checkOutputContent(repJ)
				})
			})
			Describe("POST", func() {
				BeforeEach(func() {
					args = append(args, "POST", URL[1:])
				})
				Context("JSON is piped to the command-line", func() {
					BeforeEach(func() {
						pipe.Data = reqJ
						initResponse("Do", 200, repJ)
					})
					It("Receives a result from the API", func() {
						checkRequest("POST", URL)
						checkSentContent(reqJ)
						checkOutputContent(repJ)
					})
				})
			})
			Describe("PUT", func() {
				BeforeEach(func() {
					args = append(args, "PUT", URL[1:])
				})
				Context("JSON is piped to the command-line", func() {
					BeforeEach(func() {
						pipe.Data = reqJ
						initResponse("Do", 200, repJ)
					})
					It("Receives a result from the API", func() {
						checkRequest("PUT", URL)
						checkSentContent(reqJ)
						checkOutputContent(repJ)
					})
				})
			})
			Describe("PATCH", func() {
				BeforeEach(func() {
					args = append(args, "PATCH", URL[1:])
				})
				Context("JSON is piped to the command-line", func() {
					BeforeEach(func() {
						pipe.Data = reqJ
						initResponse("Do", 204, repJ)
					})
					It("Receives a result from the API", func() {
						checkRequest("PATCH", URL)
						checkSentContent(reqJ)
					})
				})
			})
		})
	})
})
