package main

import (
	"log"
	"os"
	"os/user"

	"github.com/elasticlic/els-cli/config"
	"github.com/urfave/cli"
)

func readConfig() (*config.Config, error) {

	f := user.Current().HomeDir + "/.els/els-cli.config"

	if f, err := os.OpenFile(f); err != nil {
		return &config.Config{}, nil
	}

	defer os.Close(f)

	return config.ReadTOML(f)
}

func main() {

	var (
		c   *config.Config
		err error
	)

	if config, err = readConfig(); err != nil {
		log.Fatalf("Invalid TOML in config file %s - %s", configFile, err)
	}

	ELSCLI := NewELSCLI(cli.NewApp(), config)
	ELSCLI.Run()
}
