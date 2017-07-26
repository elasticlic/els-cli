package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os/user"
	"strconv"
	"time"

	"github.com/elasticlic/els-api-sdk-go/els"
	"github.com/elasticlic/go-utils/datetime"
	"github.com/jawher/mow.cli"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// gApp gives us access to our app from within the functions invoked by the
// jawher/mow.cli framework. There's currently no other way to avoid this other
// than to define the whole behaviour in the initialisation function as we can't
// store a context within the cli.Cmd.
var gApp *ELSCLI

// Version identifies the version of the CLI.
const Version = "0.0.3"

const (
	// APIRetryInterval governs the initial throttling of an API retry
	APIRetryInterval = time.Millisecond * 500
)

var (
	ErrNoContent      = errors.New("No Content Provided - either provide a filename or pipe content to the command")
	ErrInvalidOutput  = errors.New("Invalid output specified")
	ErrApiUnreachable = errors.New("The ELS API could not be reached. Are you connected to the internet?")
)

// ELSCLI represents our App.
type ELSCLI struct {

	// err represents a fatal error which interrupted normal execution
	fatalErr error

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

	// pw is used to obtain a password from the user.
	pw Passworder

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
	pw Passworder,
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
		pw:           pw,
		outputStream: o,
		errorStream:  e,
	}
}

// fatalError terminates the cli cleanly in the event of a usage error which
// cannot be automatically captured by the cli framework.
func (e *ELSCLI) fatalError(err error) {
	e.fatalErr = err
	log.WithFields(log.Fields{"Time": e.tp.Now(), "error": err}).Debug("Fatal Error")
	fmt.Fprintln(e.errorStream, err.Error())
}

// tryRequest makes a single attempt to do an API call
func (e *ELSCLI) tryRequest(req *http.Request) (rep *http.Response, err error) {
	if rep, err = e.apiCaller.Do(nil, req, e.profile, true); err != nil {
		log.WithFields(log.Fields{"Time": e.tp.Now(), "method": req.Method, "url": req.URL, "err": err}).Debug("Could not access API")
		return nil, ErrApiUnreachable
	}

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

	if httpMethod == "PATCH" {
		// Some ELS PATCH calls can validly have an empty body
		if bodyRC, err = e.getInputData(srcFile); err != nil {
			bodyRC = nil
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

	getBody := (e.profile.Output != OutputStatusCodeOnly) && (rep.Body != nil) && (rep.StatusCode != 204)

	var prettyJSON bytes.Buffer

	if getBody {
		data, err := ioutil.ReadAll(rep.Body)
		if err != nil {
			return err
		}

		if err := json.Indent(&prettyJSON, data, "", "\t"); err != nil {
			return err
		}
	}

	if e.profile.Output != OutputBodyOnly {
		fmt.Fprintln(e.outputStream, rep.StatusCode)
	}

	if (e.profile.Output != OutputStatusCodeOnly) && (prettyJSON.Len() > 0) {
		fmt.Fprintln(e.outputStream, prettyJSON.String())
	}

	return nil
}

// putVendor updates or creates a vendor.
func (e *ELSCLI) putVendor(vendorID string, inputFilename string) {
	if err := e.makeCall("PUT", "/vendors/"+vendorID, inputFilename); err != nil {
		e.fatalError(err)
	}
}

// getVendor retrieves the details of the given vendor.
func (e *ELSCLI) getVendor(vendorID string) {
	if err := e.get("/vendors/" + vendorID); err != nil {
		e.fatalError(err)
	}
}

// putCloudProvider updates or creates a cloud provider.
func (e *ELSCLI) putCloudProvider(cloudProviderID string, inputFilename string) {
	if err := e.makeCall("PUT", "/partners/"+cloudProviderID, inputFilename); err != nil {
		e.fatalError(err)
	}
}

// getCloudProvider retrieves the details of the given cloud provider.
func (e *ELSCLI) getCloudProvider(cloudProviderID string) {
	if err := e.get("/partners/" + cloudProviderID); err != nil {
		e.fatalError(err)
	}
}

// putRuleset defines or updates a ruleset with the given id.
func (e *ELSCLI) putRuleset(vendorId string, rulesetID string, inputFilename string) {
	URL := "/vendors/" + vendorId + "/paygRuleSets/" + rulesetID

	if err := e.makeCall("PUT", URL, inputFilename); err != nil {
		e.fatalError(err)
	}
}

// activateRuleset makes the given ruleset now the one which is used to generate
// live pricing for the vendor's products (when using Fuel).
func (e *ELSCLI) activateRuleset(vendorId string, rulesetID string) {
	URL := "/vendors/" + vendorId + "/paygRuleSets/" + rulesetID + "/activate"

	if err := e.makeCall("PATCH", URL, ""); err != nil {
		e.fatalError(err)
	}
}

// getRulesets lists all the rulesets.
func (e *ELSCLI) getRulesets(vendorId string) {

	if err := e.get("/vendors/" + vendorId + "/paygRuleSets/"); err != nil {
		e.fatalError(err)
	}
}

//createAccessKey asks for a password then makes a request to retrieve a new
// AccessKey for the user. If successful, it outputs the key as it should be
// declared in a default profile.
func (e *ELSCLI) createAccessKey(email string, expiryDays int) {

	password, err := e.pw.GetPassword()

	if err != nil {
		e.fatalError(err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(e.profile.APITimeoutSecs))
	defer cancel()

	k, statusCode, err := e.apiCaller.CreateAccessKey(ctx, email, password, false, uint(expiryDays))

	s := e.outputStream
	cr := "\n"

	if statusCode == 401 {
		fmt.Fprintln(s, "The email address or password are incorrect.")
		err = errors.New("Request Failed: (StatusCode = " + strconv.Itoa(statusCode) + ")")
	}

	if err != nil {
		e.fatalError(err)
		return
	}

	fmt.Fprintln(s, "Access Key Created - shown below in a 'default' profile.")
	fmt.Fprintln(s, "To sign API calls made by the els-cli with this access key,")
	fmt.Fprintln(s, "add the profile to ~/.els/els-cli.toml ."+cr)

	str :=
		"[profiles.default]" + cr +
			"\t[profiles.default.accessKey]" + cr +
			"\t\temail = \"" + email + `"` + cr +
			"\t\tid = \"" + string(k.ID) + `"` + cr +
			"\t\tsecretAccessKey = \"" + string(k.SecretAccessKey) + `"` + cr

	if expiryDays > 0 {
		str = str + "\t\texpiryDate = \"" + k.ExpiryDate.UTC().Format(time.RFC3339) + `"` + cr
	}

	fmt.Fprintln(e.outputStream, str)
}

// deleteAccessKey tries to delete the Access Key whose ID is kID and whose user
// is email.
func (e *ELSCLI) deleteAccessKey(email string, kID els.AccessKeyID) {
	if err := e.makeCall("DELETE", "/users/"+email+"/accessKeys/"+string(kID), ""); err != nil {
		e.fatalError(err)
	}
}

// listAccessKeys lists the AccessKeys relating to a user
func (e *ELSCLI) listAccessKeys(email string) {
	if err := e.makeCall("GET", "/users/"+email+"/accessKeys", ""); err != nil {
		e.fatalError(err)
	}
}

// doGetCommand executes a generic GET request.
func (e *ELSCLI) doGetCommand(URL string) {
	if err := e.get("/" + URL); err != nil {
		e.fatalError(err)
	}
}

// doCommand executes a generic POST, PUT or PATCH request.
func (e *ELSCLI) doCommand(method string, URL string, inputFilename string) {
	if err := e.makeCall(method, "/"+URL, inputFilename); err != nil {
		e.fatalError(err)
	}
}

// cloudProviderCommands defines commands relating to the Cloud Provider
// ('Partner') API. Note that some of these routes are only accessible to ELS
// role-holders.
func cloudProviderCommands(cpC *cli.Cmd) {
	cpC.Spec = "[CLOUDPROVIDERID]"
	cloudProviderID := cpC.StringArg("CLOUDPROVIDERID", "", "The ELS id of the cloud provider")

	cpC.Command("put", "Update or Create a cloud provider", func(c *cli.Cmd) {
		c.Spec = "[SRC]"
		content := c.StringArg("SRC", "", "The file containing the JSON defining the cloud provider")
		c.Action = func() {
			gApp.putCloudProvider(*cloudProviderID, *content)
		}
	})

	cpC.Command("get", "Get details about a cloud provider", func(c *cli.Cmd) {
		c.Action = func() {
			gApp.getCloudProvider(*cloudProviderID)
		}
	})
}

// genericCommands defines commands that allow the making of any API call except
// access key creation. All calls are els-signed (whether they need to be or
// not).
func genericCommands(gC *cli.Cmd) {

	gC.Command("GET", "Get a resource", func(c *cli.Cmd) {
		c.Spec = "URL"
		url := c.StringArg("URL", "", "The path and query string of the API call without the domain or version prefix - e.g. 'vendors/...'")
		c.Action = func() {
			gApp.doGetCommand(*url)
		}
	})
	gC.Command("PUT", "Update or Create a resource", func(c *cli.Cmd) {
		c.Spec = "URL [CONTENT]"
		url := c.StringArg("URL", "", "The path and query string of the API call without the domain or version prefix - e.g. 'vendors/...'")
		content := c.StringArg("CONTENT", "", "The file containing the JSON to be sent as the request body")
		c.Action = func() {
			gApp.doCommand("PUT", *url, *content)
		}
	})
	gC.Command("POST", "Post a resource", func(c *cli.Cmd) {
		c.Spec = "URL [CONTENT]"
		url := c.StringArg("URL", "", "The path and query string of the API call without the domain or version prefix - e.g. 'vendors/...'")
		content := c.StringArg("CONTENT", "", "The file containing the JSON to be sent as the request body")
		c.Action = func() {
			gApp.doCommand("POST", *url, *content)
		}
	})
	gC.Command("PATCH", "Patch a resource", func(c *cli.Cmd) {
		c.Spec = "URL [CONTENT]"
		url := c.StringArg("URL", "", "The path and query string of the API call without the domain or version prefix - e.g. 'vendors/...'")
		content := c.StringArg("CONTENT", "", "The file containing the JSON to be sent as the request body")
		c.Action = func() {
			gApp.doCommand("PATCH", *url, *content)
		}
	})
}

// vendorCommands defines commands relating to the Vendor API. Note that some
// of these routes are only accessible to ELS role-holders.
func vendorCommands(vendorC *cli.Cmd) {
	vendorC.Spec = "VENDORID"
	vendorID := vendorC.StringArg("VENDORID", "", "The ELS id of the vendor")

	vendorC.Command("put", "Update or Create a vendor", func(c *cli.Cmd) {
		c.Spec = "[SRC]"
		content := c.StringArg("SRC", "", "The file containing the JSON defining the vendor")
		c.Action = func() {
			gApp.putVendor(*vendorID, *content)
		}
	})

	vendorC.Command("get", "Get details about a vendor", func(c *cli.Cmd) {
		c.Action = func() {
			gApp.getVendor(*vendorID)
		}
	})

	vendorC.Command("rulesets", "Manage Fuel Charging Rulesets - used to generate pricing for Fuel.", func(rulesetsC *cli.Cmd) {
		rulesetID := rulesetsC.StringArg("RULESETID", "", "The ID of the ruleset")

		rulesetsC.Command("put", "Create or update a Fuel Charging Ruleset - note you cannot update an activated (live) ruleset.", func(c *cli.Cmd) {
			c.Spec = "[SRC]"
			content := c.StringArg("SRC", "", "The file containing the JSON defining the ruleset")
			c.Action = func() {
				gApp.putRuleset(*vendorID, *rulesetID, *content)
			}
		})

		rulesetsC.Command("get", "List all the rulesets", func(c *cli.Cmd) {
			c.Action = func() {
				gApp.getRulesets(*vendorID)
			}
		})

		rulesetsC.Command("activate", "Activate Fuel Charging Ruleset - i.e. it will be the ruleset currently used to define prices", func(c *cli.Cmd) {
			c.Action = func() {
				gApp.activateRuleset(*vendorID, *rulesetID)
			}
		})
	})
}

// userCommands defines the commands relating to the User API.
func userCommands(userC *cli.Cmd) {
	email := userC.StringArg("EMAIL", "", "The email address of the user")

	userC.Command("accessKeys", "Manage Access Keys", func(accessKeysC *cli.Cmd) {
		accessKeysC.Command("create", "Create a new API Access Key", func(c *cli.Cmd) {
			c.Spec = "[EXPIRYDAYS]"
			expiryDays := c.IntArg("EXPIRYDAYS", 30, "Number of days before expiry.")
			c.Action = func() {
				gApp.createAccessKey(*email, *expiryDays)
			}
		})
		accessKeysC.Command("delete", "Delete an API Access Key", func(c *cli.Cmd) {
			keyId := c.StringArg("ACCESSKEYID", "", "The ID of the Access Key to be deleted")
			c.Action = func() {
				gApp.deleteAccessKey(*email, els.AccessKeyID(*keyId))
			}
		})
		accessKeysC.Command("list", "List API Access Keys", func(c *cli.Cmd) {
			c.Action = func() {
				gApp.listAccessKeys(*email)
			}
		})
	})
}

// initProfile identifies which profile from the config should be used for
// default values (if any is set). An optional output o can be used to override
// the default output in the profile.
func (e *ELSCLI) initProfile(p string, o string) (err error) {

	e.profile, err = e.config.Profile(p)

	// We don't expect people to have a config file so if the default profile
	// doesn't exist in the config, don't flag the error.
	if err != nil && p != "default" {
		return ErrProfileNotFound
	}

	// overrides
	if o != "" {
		e.profile.Output = o
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
func (e *ELSCLI) init() error {
	if err := e.initLog(); err != nil {
		e.fatalError(err)
		return err
	}
	// store our app for access from the framework callbacks later:
	gApp = e

	a := e.fApp

	a.Version("v version", Version)
	prof := a.String(cli.StringOpt{
		Name:   "p profile",
		Value:  "default",
		Desc:   "specify which profile in ~/.els/els-cli.config supplies credentials",
		EnvVar: "ELSCLI_PROFILE",
	})

	output := a.String(cli.StringOpt{
		Name:   "o output",
		Value:  "",
		Desc:   "Overrides the output format defined in the profile: Must be: wholeResponse|bodyOnly|statusCodeOnly",
		EnvVar: "ELSCLI_OUTPUT",
	})
	a.Before = func() {
		e.initProfile(*prof, *output)
	}

	a.Command("users", "User API", userCommands)
	a.Command("vendors", "Vendor API", vendorCommands)
	a.Command("cloud-providers", "Cloud Provider API", cloudProviderCommands)
	a.Command("do", "Make any call to the API", genericCommands)

	return nil
}

// Run parses the command line arguments and tries to identify and execute a
// command. It returns any fatal errors - e.g. configuration errors or usage
// errors. It does not return an error if a request failed because of ELS
// permissions (for example)
func (e *ELSCLI) Run(cliArgs []string) error {

	if err := e.init(); err != nil {
		return err
	}

	e.fApp.Run(cliArgs)

	return e.fatalErr
}
