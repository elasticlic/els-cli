package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/elasticlic/els-api-sdk-go/els"
	"github.com/elasticlic/go-utils/datetime"
	"github.com/jawher/mow.cli"
	"github.com/spf13/afero"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// gApp gives us access to our app from within the functions invoked by the
// jawher/mow.cli framework. There's currently no other way to avoid this other
// than to define the whole behaviour in the initialisation function as we can't
// store a context within the cli.Cmd.
var gApp *ELSCLI

const (
	// APIRetryInterval governs the initial throttling of an API retry
	APIRetryInterval = time.Millisecond * 500
)

var (
	ErrNoContent      = errors.New("No Content Provided - either provide a filename or pipe content to the command")
	ErrApiUnreachable = errors.New("The ELS API could not be reached. Are you connected to the internet?")
)

// Pipe is the interface which defines methods operating on data piped to
// the command via the command-line.
type Pipe interface {
	Reader() (io.ReadCloser, error)
}

// CLIPipe is used to read data piped to the els-cli from the commmand-line.
type CLIPipe struct{}

func NewCLIPipe() *CLIPipe {
	return &CLIPipe{}
}

// Reader implements interface Pipe and returns a Reader which can be used to
// read the data passed to the els-cli via a command-line pipe.
func (p *CLIPipe) Reader() (io.ReadCloser, error) {
	info, err := os.Stdin.Stat()

	if err != nil {
		return nil, err
	}

	if (info.Mode()&os.ModeCharDevice) != 0 || info.Size() == 0 {
		// no data from a pipe - ignore
		return nil, ErrNoContent
	}

	return os.Stdin, nil
}

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
	profile *Profile

	// fs is an abstraction of the filesystem which makes it easier to test.
	fs afero.Fs

	// pipe allows access to data piped from the command-line to the app.
	pipe Pipe

	// outputStream is the stream to which results are written.
	outputStream io.Writer

	// errorStream is the stream to which errors are written.
	errorStream io.Writer

	// tp provides time for the app.
	tp datetime.TimeProvider
}

// NewELSCLI creates a new instance of the ELS CLI App. Call Run() to execute
// the App.
func NewELSCLI(
	cliApp *cli.Cli,
	c *Config,
	cFile string,
	tp datetime.TimeProvider,
	fs afero.Fs,
	a els.APICaller,
	p Pipe,
	o io.Writer,
	e io.Writer) *ELSCLI {
	return &ELSCLI{
		fApp:         cliApp,
		config:       c,
		configFile:   cFile,
		tp:           tp,
		apiCaller:    a,
		fs:           fs,
		pipe:         p,
		outputStream: o,
		errorStream:  e,
	}
}

// fatalError terminates the cli cleanly in the event of a usage error which
// cannot be automatically captured by the cli framework.
func (e *ELSCLI) fatalError(err error) {
	log.WithFields(log.Fields{"Time": e.tp.Now(), "Error": err}).Debug("Fatal Error")
	cli.Exit(-1)
}

// tryRequest makes a single attempt to do an API call
func (e *ELSCLI) tryRequest(req *http.Request) (rep *http.Response, err error) {
	if rep, err = e.apiCaller.Do(nil, req, e.profile, true); err != nil {
		log.WithFields(log.Fields{"Time": e.tp.Now(), "method": req.Method, "url": req.URL, "err": err}).Debug("Could not access API")
		return nil, ErrApiUnreachable
	}

	log.WithFields(log.Fields{"Time": e.tp.Now(), "method": req.Method, "url": req.URL, "statusCode": rep.StatusCode}).Debug("API Call throttled by ELS")
	return rep, nil
}

// doRequest attempts the given request, retrying if necessary.
func (e *ELSCLI) doRequest(req *http.Request) (rep *http.Response, err error) {
	for t := 0; t < e.profile.MaxAPITries; t++ {
		rep, err = e.tryRequest(req)

		if (err == nil) && (rep.StatusCode != http.StatusTooManyRequests) {
			break
		}
	}

	return
}

// getJSON returns a ReadCloser which will supply the JSON for the API
// call - either from srcFile or, if not defined, from data piped into the
// command.
func (e *ELSCLI) getInputData(srcFile string) (io.ReadCloser, error) {

	if srcFile == "" {
		return e.pipe.Reader()
	}

	return e.fs.Open(srcFile)
}

// get makes a GET call to the given URL, where URL is relative to the API root
// e.g. "/vendors".
func (e *ELSCLI) get(URL string) error {
	return e.makeCall("GET", URL, "")
}

// makeCall executes an API call whose body will be set to the contents of the
// given file, or, if no file is given, data piped to the command. The URL is
// relative to the API root - e.g. "/vendors".
func (e *ELSCLI) makeCall(httpMethod string, URL string, srcFile string) (err error) {

	var (
		bodyRC io.ReadCloser
		rep    *http.Response
	)

	if (httpMethod == "POST") || (httpMethod == "PUT") {
		if bodyRC, err = e.getInputData(srcFile); err != nil {
			return err
		}
	}

	req, err := http.NewRequest(httpMethod, URL, bodyRC)
	if err != nil {
		log.WithFields(log.Fields{"Time": e.tp.Now(), "url": URL, "error": err}).Debug("putRequest")
		return err
	}

	if rep, err = e.doRequest(req); err != nil {
		return err
	}

	return e.writeResponse(rep)
}

// writeResponse outputs the requested components of the received response.
func (e *ELSCLI) writeResponse(rep *http.Response) error {

	if rep.Body != nil {
		defer rep.Body.Close()
	}

	getBody := (e.profile.Output != OutputStatusCodeOnly) && (rep.Body != nil) && rep.ContentLength > 0

	var prettyJSON bytes.Buffer

	if getBody {
		JSON, err := ioutil.ReadAll(rep.Body)
		if err != nil {
			return err
		}
		if err := json.Indent(&prettyJSON, JSON, "", "\t"); err != nil {
			return err
		}
	}

	if e.profile.Output != OutputBodyOnly {
		fmt.Fprintln(e.outputStream, rep.StatusCode)
	}

	if (e.profile.Output != OutputStatusCodeOnly) && (prettyJSON.Len() > 0) {
		fmt.Fprintln(e.outputStream, prettyJSON)
	}

	return nil
}

// putVendor updates or creates a vendor.
func (e *ELSCLI) putVendor(vendorId string, inputFilename string) {
	if err := e.makeCall("PUT", "/vendors/"+vendorId, inputFilename); err != nil {
		e.fatalError(err)
	}
}

// getVendor retrieves the details of the given vendor.
func (e *ELSCLI) getVendor(vendorId string) {
	if err := e.get("/vendors/" + vendorId); err != nil {
		e.fatalError(err)
	}
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
func (e *ELSCLI) initLog() error {

	u, err := user.Current()
	if err != nil {
		return err
	}

	log.SetOutput(&lumberjack.Logger{
		Filename:   u.HomeDir + "/.els/els-cli.log",
		MaxSize:    10, // megabytes
		MaxBackups: 3,
		MaxAge:     28, //days
	})
	log.SetLevel(log.DebugLevel)

	return nil
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
