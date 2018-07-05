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
		eulaPeriod          = "month"
		year                = "2018"
		month               = "7"
		rulesetID           = "aRuleset"
		URL                 = "/a/path/and?querystring"
		cursor              = "aCursor"
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

			// checkOutputString checks if the output content is the string
			// given.
			checkOutputString = func(str string) {
				Expect(errS.String()).To(BeZero())
				if str != "" {
					Expect(outS.String()).To(Equal(str))
				}
			}

			// checkOutputJSON checks if the els-cli emitted the JSON response
			// body received from the ELS.
			checkOutputJSON = func(json string) {
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
					checkOutputJSON(repJ)
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
						checkOutputJSON(repJ)
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
						checkOutputJSON(repJ)
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
					checkOutputJSON(repJ)
				})
			})
			Describe("list-rulesets", func() {
				BeforeEach(func() {
					args = append(args, "list-rulesets")
					initResponse("Do", 200, repJ)
				})
				It("Receives a result from the API", func() {
					checkRequest("GET", "/vendors/"+vendorID+"/paygRuleSets")
					checkOutputJSON(repJ)
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
						checkOutputJSON(repJ)
					})
				})
				Describe("get", func() {
					BeforeEach(func() {
						args = append(args, rulesetID, "get")
						initResponse("Do", 200, repJ)
					})
					It("Receives a result from the API", func() {
						checkRequest("GET", "/vendors/"+vendorID+"/paygRuleSets/"+rulesetID)
						checkOutputJSON(repJ)
					})
				})
				Describe("activate", func() {
					BeforeEach(func() {
						args = append(args, rulesetID, "activate")
						initResponse("Do", 204, "")
					})
					It("Receives a result from the API", func() {
						checkRequest("PATCH", "/vendors/"+vendorID+"/paygRuleSets/"+rulesetID+"/activate")
						checkOutputJSON("")
					})
				})
			})
			Describe("get-eula-license-infringements", func() {
				BeforeEach(func() {
					args = append(args, "get-eula-license-infringements")
				})
				Context("The Year and Month for the report is given", func() {
					BeforeEach(func() {
						args = append(args, year, month)
					})
					Context("There are multiple pages of infringements", func() {
						BeforeEach(func() {
							rJSON1 := `
								{
									"cursor": "` + cursor + `",
									"customerInfringements": [
										{
											"elsCustomerId": "elsCustomerID1",
											"vendorCustomerId": "vendorCustomerID1",
											"infringements": [
												{
													"eulaPeriod": "` + eulaPeriod + `",
													"year": ` + year + `,
													"month": ` + month + `,
													"vendorId": "` + vendorID + `",
													"eulaPolicyId": "eulaPolicyID1",
													"featureId": "featureID1",
													"licenceSetId": "licenceSetID1",
													"licenceIndex": 2,
													"numUsers": 5
												},
												{
													"eulaPeriod": "` + eulaPeriod + `",
													"year": ` + year + `,
													"month": ` + month + `,
													"vendorId": "` + vendorID + `",
													"eulaPolicyId": "eulaPolicyID2",
													"featureId": "featureID2",
													"licenceSetId": "licenceSetID2",
													"licenceIndex": 4,
													"numUsers": 6
												}
											]
										}
									]
								}
							`
							ac.AddExpectedCall("Do", em.APICall{
								ACRep: em.ACRep{
									Rep: em.HTTPResponse(200, rJSON1),
								},
							})
							rJSON2 := `
								{
									"cursor": "",
									"customerInfringements": [
										{
											"elsCustomerId": "elsCustomerID2",
											"vendorCustomerId": "vendorCustomerID2",
											"infringements": [
												{
													"eulaPeriod": "` + eulaPeriod + `",
													"year": ` + year + `,
													"month": ` + month + `,
													"vendorId": "` + vendorID + `",
													"eulaPolicyId": "eulaPolicyID3",
													"featureId": "featureID3",
													"licenceSetId": "licenceSetID3",
													"licenceIndex": 0,
													"numUsers": 1
												}
											]
										}
									]
								}
							`
							ac.AddExpectedCall("Do", em.APICall{
								ACRep: em.ACRep{
									Rep: em.HTTPResponse(200, rJSON2),
								},
							})
						})
						It("returns the expected CSV", func() {

							expectedCSV :=
								"elsCustomerID,vendorCustomerID,eulaPeriod,year,month,eulaPolicyID,featureID,licenseSetID,licenseIndex,numUsers\n" +
									"elsCustomerID1,vendorCustomerID1,month,2018,7,eulaPolicyID1,featureID1,licenceSetID1,2,5\n" +
									"elsCustomerID1,vendorCustomerID1,month,2018,7,eulaPolicyID2,featureID2,licenceSetID2,4,6\n" +
									"elsCustomerID2,vendorCustomerID2,month,2018,7,eulaPolicyID3,featureID3,licenceSetID3,0,1\n"

							checkOutputString(expectedCSV)
						})
					})
					Context("There are no infringements", func() {
						BeforeEach(func() {
							rJSON1 := `
								{
									"cursor": "` + cursor + `",
									"customerInfringements": []
								}
							`
							ac.AddExpectedCall("Do", em.APICall{
								ACRep: em.ACRep{
									Rep: em.HTTPResponse(200, rJSON1),
								},
							})
							rJSON2 := `
								{
									"cursor": "",
									"customerInfringements": []
								}
							`
							ac.AddExpectedCall("Do", em.APICall{
								ACRep: em.ACRep{
									Rep: em.HTTPResponse(200, rJSON2),
								},
							})
						})
						It("returns the expected CSV", func() {

							expectedCSV := "elsCustomerID,vendorCustomerID,eulaPeriod,year,month,eulaPolicyID,featureID,licenseSetID,licenseIndex,numUsers\n"

							checkOutputString(expectedCSV)
						})
					})
					Context("An unexpected response is received", func() {
						BeforeEach(func() {
							ac.AddExpectedCall("Do", em.APICall{
								ACRep: em.ACRep{
									Rep: em.HTTPResponse(500, ""),
								},
							})
						})
						It("reports an unexpected error", func() {
							Expect(fatalErr).To(Equal(cli.ErrUnexpectedResponse))
						})
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
						checkOutputJSON(repJ)
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
					checkOutputJSON(repJ)
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
					checkOutputJSON(repJ)
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
						checkOutputJSON(repJ)
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
						checkOutputJSON(repJ)
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
			Describe("DELETE", func() {
				BeforeEach(func() {
					args = append(args, "DELETE", URL[1:])
					initResponse("Do", 200, repJ)
				})

				It("Issues the expected request", func() {
					checkRequest("DELETE", URL)
				})
			})
		})
	})
})
