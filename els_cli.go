package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/elasticlic/els-api-sdk-go/els"
	"github.com/jawher/mow.cli"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// gApp gives us access to our app from within the functions invoked by the
// jawher/mow.cli framework. There's currently no other way to avoid this other
// than to define the whole behaviour in the initialisation function as we can't
// store a context within the cli.Cmd.
var gApp *ELSCLI

var (
	ErrNoContent = errors.New("No Content Provided - either provide a filename or pipe content to the command")
)

// ELSCLI represents our App.
type ELSCLI struct {

	// fApp is a framework which parses the commandline.
	fApp *cli.Cli

	// config is the data read from the config file.
	config *Config

	// configFile is the location of the config file read in.
	configFile string

	// apiCaller is used to request access keys and make signed API calls to the
	// ELS.
	apiCaller els.APICaller

	// profile is the collection of properties which may be needed to make the
	// API call. Optionally they can be prefilled by a profile from the config
	// file, selected via --profile. Additionally, individual properties can
	// be set on with flags the commandline or via environment variables.
	profile Profile
}

// NewELSCLI creates a new instance of the ELS CLI App. Call Run() to execute
// the App.
func NewELSCLI(cliApp *cli.Cli, c *Config, cFile string, a els.APICaller) *ELSCLI {
	return &ELSCLI{
		fApp:       cliApp,
		config:     c,
		configFile: cFile,
		apiCaller:  a,
	}
}

// fatalError terminates the cli cleanly in the event of a usage error which
// cannot be automatically captured by the cli framework.
func (e *ELSCLI) fatalError(err error) {
	log.WithFields(log.Fields{"Time": time.Now(), "Error": err}).Debug("Fatal Error")
	cli.Exit(-1)
}

// If data was piped into the app, it returns the bytes, otherwise nil.
func (e *ELSCLI) readCLIPipeData() (bytes []byte) {
	info, _ := os.Stdin.Stat()
	if info.Size() > 0 {
		bytes, _ := ioutil.ReadAll(os.Stdin)
	}
	return
}

// jsonError reports the first error (if any) of the supposed JSON passed.
func (e *ELSCLI) jsonError(j []byte) bool {
	var js map[string]interface{}
	return json.Unmarshal(j, &js)
}

// putRequest initialises an HTTP request whose body will be set to the contents
// of the given file. If no file is given, it checks if data was piped to the
// command, and if so, uses that instead. If the body cannot be identified, then
// a fatal error is generated.
func (e *ELSCLI) putRequest(url string, srcFile string) http.Request {

	b, err := ioutil.ReadFile(srcFile)

	if err != nil {
		b := e, readCLIPipeData()
	}

	if b == nil {
		e.fatalError(ErrNoContent)
	}

	// Validate input - must be JSON.
	jErr := e.jsonError(b)
	if jErr != nil {
		e.fatalError(jErr)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	if err != nil {
		log.WithFields(log.Fields{"Time": time.Now(), "url": url, "json": string(jsonB)}).Debug("putRequest")
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
}

func (e *ELSCLI) putVendor(vendorId string, inputFilename string) {
	r := e.putRequest("/vendors/"+vendorId, inputFilename)
}

func (e *ELSCLI) getVendor(vendorId string, outputFilename string) {

	iowriter

	if err != nil {
		e.fatalError(err)
	}

	e.apiCaller.Do(nil)
}

// vendorCommands defines the vendor subcommands.
func vendorCommands(vc *cli.Cmd) {
	vendorId := vc.StringArg("VENDORID", "", "The ELS id of the vendor")

	vc.Command("put", "Update or Create a vendor", func(c *cli.Cmd) {
		c.Spec = "[SRC]"
		content := c.StringArg("SRC", "", "The file containing the JSON defining the vendor")
		c.Action = func() {
			gApp.putVendor(*vendorId, *content)
		}
	})

	vc.Command("get", "Get a vendor", getVendor)
}

func getVendor(c *cli.Cmd) {
	//c.Spec = "VendorID..."
	//	vendor := c.StringsArg("VendorID", nil, "")
	c.Action = func() {
		fmt.Println("Get vendor called")
	}
}

// initProfile identifies which profile from the config should be used for
// default values (if any is set)
func (e *ELSCLI) initProfile(p string) (err error) {
	fmt.Println("CALLED INIT PROFILE", p)

	e.profile, err = e.config.Profile(p)

	// We don't expect people to have a config file so if the default profile
	// doesn't exist in the config, don't flag the error.
	if err != nil && p != "default" {
		return ErrProfileNotFound
	}

	return nil
}

// initLog configures logrus to create rotating logs within the user's .els
// directory.
func (e *ELSCLI) initLog() {

	u, err := user.Current()
	if err != nil {
		return "", err
	}

	log.SetOutput(&lumberjack.Logger{
		Filename:   u.HomeDir + "/.els/els-cli.log",
		MaxSize:    10, // megabytes
		MaxBackups: 3,
		MaxAge:     28, //days
	})
	log.SetLevel(log.DebugLevel)
}

// init sets up the app prior to parsing the commandline.
func (e *ELSCLI) init() {
	if err := e.initLog(); err != nil {
		e.fatalError(err)
	}
	// store our app for access from the framework callbacks later:
	gApp = e

	a := e.fApp

	a.Version("v version", "0.0.1d")
	prof := a.String(cli.StringOpt{
		Name:   "p profile",
		Value:  "default",
		Desc:   "specify which profile in ~/.els/els-cli.config supplies credentials",
		EnvVar: "ELSCLI_PROFILE",
	})
	a.Before = func() {
		e.initProfile(*prof)
	}

	a.Command("vendor", "Vendor API", vendorCommands)

}

// Run parses the command line arguments and tries to identify and execute a
// command.
func (e *ELSCLI) Run(cliArgs []string) {
	e.init()
	e.fApp.Run(cliArgs)
}
