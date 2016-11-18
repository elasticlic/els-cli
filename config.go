package main

import (
	"errors"
	"io"

	"github.com/BurntSushi/toml"
	"github.com/elasticlic/els-api-sdk-go/els"
)

// Errors relating to configuration.
var (
	ErrProfileNotFound = errors.New("Profile not found")
)

// Profile represents a named set of defaults.
type Profile struct {
	// AccessKey is used to sign API calls. An Access Key can be generated using
	// the CLI  (TODO - describe process).
	AccessKey els.AccessKey
}

// Config represents a parsed configuration which provides defaults for commands
// issued with the els-cli.
type Config struct {
	// Profiles stores all the profiles read from the TOML file, indexed by
	// profile ID.
	Profiles map[string]Profile
}

// Profile returns the profile matching the given ID, or an empty profile if
// not found.
func (c *Config) Profile(profileID string) (Profile, error) {

	if p, ok := c.Profiles[profileID]; ok {
		return p, nil
	}
	return Profile{}, ErrProfileNotFound
}

// ReadTOML returns a config object initialised with the TOML data provided by
// the reader.
func ReadTOML(r io.Reader) (c *Config, err error) {
	c = &Config{}

	_, err = toml.DecodeReader(r, c)
	return c, err
}
