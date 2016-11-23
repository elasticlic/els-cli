package main_test

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/elasticlic/els-api-sdk-go/els"
	cli "github.com/elasticlic/els-cli"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config Test Suite", func() {

	var (
		err error
	)

	Describe("Profile", func() {
		var (
			sut *cli.Profile
			k   els.AccessKey
			req *http.Request
			now = time.Now()
		)
		BeforeEach(func() {
			sut = cli.NewProfile()
		})

		Describe("NewProfile", func() {
			It("Creates the expected default struct", func() {
				Expect(sut).To(BeEquivalentTo(cli.Profile{
					MaxAPITries: 2,
					Output:      cli.OutputWhole,
				}))
			})
		})
		Describe("Sign", func() {
			BeforeEach(func() {
				req, err = http.NewRequest("POST", "http:/a/url", nil)
				Expect(err).To(BeNil())
			})
			JustBeforeEach(func() {
				err = sut.Sign(req, now)
			})
			Context("The key is not set", func() {
				It("returns ErrNoAccessKey", func() {
					Expect(err).To(Equal(els.ErrNoAccessKey))
				})
			})
			Context("The key is set", func() {
				It("adds expected headers", func() {
					Expect(req.Header.Get("Authorization")).NotTo(BeZero())
					Expect(req.Header.Get("X-Els-Date")).NotTo(BeZero())
					Expect(req.Header.Get("Content-Type")).To(Equal(els.RequiredContentType))
				})
			})
		})
	})

	Describe("Config", func() {
		var (
			sut       *cli.Config
			p         *cli.Profile
			profileID = "aProfile"
		)
		BeforeEach(func() {
			sut = &cli.Config{
				Profiles: make(map[string]cli.Profile),
			}
			sut.Profiles[profileID] = cli.Profile{AccessKey: els.AccessKey{ID: "123"}}
		})

		Describe("Profile", func() {
			JustBeforeEach(func() {
				p, err = sut.Profile(profileID)
			})
			Context("An existing profile is requested", func() {
				It("returns the profile", func() {
					Expect(err).To(BeNil())
					Expect(p.AccessKey.ID).To(BeEquivalentTo("123"))
				})
			})
			Context("A non-existing profile is requested", func() {
				BeforeEach(func() {
					profileID = "unknownProfile"
				})
				It("returns ErrProfileNotFound", func() {
					Expect(err).To(Equal(cli.ErrProfileNotFound))
				})
			})
		})

		Describe("ReadTOML", func() {
			var (
				r    io.Reader
				c    *cli.Config
				toml string
			)
			JustBeforeEach(func() {
				r = strings.NewReader(toml)
				c, err = cli.ReadTOML(r)
			})
			Context("Invalid TOML is passed", func() {
				BeforeEach(func() {
					toml = "[unclosed bracket"
				})
				It("returns an error", func() {
					Expect(err).NotTo(BeNil())
				})
			})
			Context("Valid TOML is passed", func() {
				BeforeEach(func() {
					toml = `
                        [profiles.default]
                            [profiles.default.accessKey]
                                id = "elsID1"
                                secretAccessKey = "secretAccessKey1"
                                email = "email1@example.com"

                        [profiles.another]
                            [profiles.another.accessKey]
                                id = "elsID2"
                                secretAccessKey = "secretAccessKey2"
                                email = "email2@example.com"
                    `
				})
				It("returns the expected values", func() {
					Expect(err).To(BeNil())
					p, err = c.Profile("default")
					Expect(err).To(BeNil())
					Expect(p).To(BeEquivalentTo(cli.Profile{
						AccessKey: els.AccessKey{
							ID:              "elsID1",
							SecretAccessKey: "secretAccessKey1",
							Email:           "email1@example.com",
						},
					}))
					p, err = c.Profile("another")
					Expect(err).To(BeNil())
					Expect(p).To(BeEquivalentTo(cli.Profile{
						AccessKey: els.AccessKey{
							ID:              "elsID2",
							SecretAccessKey: "secretAccessKey2",
							Email:           "email2@example.com",
						},
					}))
				})
			})
		})
	})
})
