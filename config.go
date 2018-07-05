package main

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/elasticlic/els-api-sdk-go/els"
)

// Errors relating to configuration.
var (
	ErrProfileNotFound = errors.New("Profile not found")
)

// Constants representing a specific output type
const (
	OutputWhole          = "wholeResponse"
	OutputBodyOnly       = "bodyOnly"
	OutputStatusCodeOnly = "statusCodeOnly"
)

// Profile represents a named set of defaults.
type Profile struct {
	// AccessKey is used to sign API calls. An Access Key can be generated using
	// the CLI  (TODO - describe process).
	AccessKey els.AccessKey

	// MaxAPITries determines how many times to try an API call before giving
	// up.
	MaxAPITries int

	// Output identifies what part of the response to output
	Output string

	// APITimeoutSecs defines how long to wait for a reply before giving up.
	APITimeoutSecs int
}

// Sign implements els.Signer and signs the given request with the access key.
func (p *Profile) Sign(r *http.Request, now time.Time) error {
	s, err := els.NewAPISigner(&p.AccessKey)
	if err != nil {
		return err
	}

	return s.Sign(r, now)
}

// SetDefaults updates any invalid zero-values to their defaults.
func (p *Profile) SetDefaults() {
	if p.MaxAPITries == 0 {
		p.MaxAPITries = 2
	}

	if p.Output == "" {
		p.Output = OutputWhole
	}

	if p.APITimeoutSecs == 0 {
		p.APITimeoutSecs = 30
	}
}

// NewProfile creates a default profile containing default settings.
func NewProfile() *Profile {
	p := &Profile{}
	p.SetDefaults()
	return p
}

// Config represents a parsed configuration which provides defaults for commands
// issued with the els-cli.
type Config struct {
	// Profiles stores all the profiles read from the TOML file, indexed by
	// profile ID.
	Profiles map[string]*Profile
}

// Profile returns the profile matching the given ID, or an empty profile if
// not found.
func (c *Config) Profile(profileID string) (*Profile, error) {

	if p, ok := c.Profiles[profileID]; ok {
		return p, nil
	}
	return NewProfile(), ErrProfileNotFound
}

// ReadTOML returns a config object initialised with the TOML data provided by
// the reader.
func ReadTOML(r io.Reader) (c *Config, err error) {
	c = &Config{}

	_, err = toml.DecodeReader(r, c)

	for _, p := range c.Profiles {
		p.SetDefaults()
	}

	return c, err
}
