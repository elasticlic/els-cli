package config

import (
	"io"
	"strings"

	"github.com/elasticlic/els-api-sdk-go/els"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config Test Suite", func() {

	var (
		err    error
		custID = "aCustomerId"
	)

	Describe("Config", func() {
		var (
			sut       *Config
			p         Profile
			profileID = "aProfile"
		)
		BeforeEach(func() {
			sut = &Config{
				Profiles: make(map[string]Profile),
			}
			sut.Profiles[profileID] = Profile{CustomerID: custID}
		})

		Describe("Profile", func() {
			JustBeforeEach(func() {
				p, err = sut.Profile(profileID)
			})
			Context("An existing profile is requested", func() {
				It("returns the profile", func() {
					Expect(err).To(BeNil())
					Expect(p.CustomerID).To(Equal(custID))
				})
			})
			Context("A non-existing profile is requested", func() {
				BeforeEach(func() {
					profileID = "unknownProfile"
				})
				It("returns ErrProfileNotFound", func() {
					Expect(err).To(Equal(ErrProfileNotFound))
				})
			})
		})

		Describe("ReadTOML", func() {
			var (
				r    io.Reader
				c    *Config
				toml string
			)
			JustBeforeEach(func() {
				r = strings.NewReader(toml)
				c, err = ReadTOML(r)
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
                            customerID = "customerA"
                            vendorID = "vendorA"
                            cloudProviderID = "cloudProviderA"
                            [profiles.default.accessKey]
                                id = "elsID1"
                                secretAccessKey = "secretAccessKey1"
                                email = "email1@example.com"

                        [profiles.another]
                            customerID = "customerB"
                            vendorID = "vendorB"
                            cloudProviderID = "cloudProviderB"
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
					Expect(p).To(BeEquivalentTo(Profile{
						AccessKey: els.AccessKey{
							ID:              "elsID1",
							SecretAccessKey: "secretAccessKey1",
							Email:           "email1@example.com",
						},
						CustomerID:      "customerA",
						VendorID:        "vendorA",
						CloudProviderID: "cloudProviderA",
					}))
					p, err = c.Profile("another")
					Expect(err).To(BeNil())
					Expect(p).To(BeEquivalentTo(Profile{
						AccessKey: els.AccessKey{
							ID:              "elsID2",
							SecretAccessKey: "secretAccessKey2",
							Email:           "email2@example.com",
						},
						CustomerID:      "customerB",
						VendorID:        "vendorB",
						CloudProviderID: "cloudProviderB",
					}))
				})
			})
		})
	})
})
