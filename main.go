package main

import (
	"log"
	"os"
	"os/user"
	"time"

	"github.com/elasticlic/els-api-sdk-go/els"
	"github.com/elasticlic/go-utils/datetime"
	"github.com/jawher/mow.cli"
	"github.com/spf13/afero"
)

// configFile identifies the expected path to the user's config file.
func configFile() (string, error) {

	u, err := user.Current()
	if err != nil {
		return "", err
	}

	return u.HomeDir + "/.els/els-cli.toml", nil
}

// readConfig attempts to identify and read the current user's els-cli.config
// file which is used to configure defaults that will be used if not passed on
// the commandline with a command.
func readConfig() (*Config, string) {

	cFile, err := configFile()
	if err != nil {
		return &Config{}, ""
	}

	f, err := os.Open(cFile)

	if err != nil {
		return &Config{}, ""
	}

	defer f.Close()

	c, err := ReadTOML(f)
	if err != nil {
		log.Fatalf("Invalid TOML in config file %s:\n%s", cFile, err)
	}

	return c, cFile
}

func main() {
	c, cFile := readConfig()
	ca := cli.App("els-cli", "Make API calls to Elastic Licensing")
	tp := datetime.NewNowTimeProvider()
	a := els.NewEDAPICaller(nil, tp, time.Second*5, "")
	fs := afero.NewOsFs()
	p := NewCLIPipe()
	pw := NewHiddenPassworder(os.Stdout)

	ELSCLI := NewELSCLI(ca, c, cFile, tp, fs, a, p, pw, os.Stdout, os.Stderr)

	ELSCLI.Run(os.Args)
}
