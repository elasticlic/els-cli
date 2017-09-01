# els-cli
A commandline tool for interacting with the Elastic Licensing API. This tool
complements the online dashboards and direct use of the API.

## Terminology

**ELS** - *Elastic Licensing Service* - a cloud-based service which implements
the Elastic API - an HTTP REST Service.

**User Account** - Users are identified by *email address*, authenticated by
password. Most API calls must be made by a user, identified not by email address
and password but by *Access Key*.

**Entity** - An entity is an account in Elastic Licensing. There are four types:
* customer account
* vendor account
* a cloud
* the Elastic Licensing Service itself.

**Role** - A user may hold one or more *roles* for one or more entities. For
example, a person may be the primary contact both for their own indvidual
customer account, and the customer account used by their place of work. When
using the API or the dashboards, the roles held by the signed-in user determine
what the user is permitted to see and do.

**Elastic App** - An App which includes the **Elastic Client Library**, and so
can use Elastic Licensing to control access by users.

**ELS-sign** - means to calculate and add a signature to an HTTP request so that
the ELS can be sure who made the API call.

**Access Key** - A key associated with a user, consisting of a public string
called the *AccessKeyID*, and a secret string called the *SecretAccessKey*.
Access Keys are used to ELS-sign an API call. Access Keys can optionally expire,
can be deleted at any time and avoid exposing personal passwords. When an
Elastic App is used an automated context (e.g. a render job), an AccessKeyID and
SecretAccessKey can be presented on the commandline to avoid needing a user to
provide a password.

Access Keys can be created either via a special API call, or using the els-cli.
See below for details.

**els-cli.toml** - a config file used by els-cli to provide certain values
and defaults. The config contains one or more *Profiles*. and should be placed
at the following location:

    ~/.els/els-cli.toml

The contents are in [TOML](https://github.com/toml-lang/toml) format.

**Profile** - defines which Access Key will ELS-sign API calls, and other
optional parameters which affect the behaviour of the els-cli. `els-cli.config`
contains one or more profiles identified by *ProfileID*, and you can specify
which profile to use by invoking els-cli as follows:

    els-cli --profile "bob" [rest of command]

If you don't specify a profile, the els-cli will use **default** profile. Here
is an example of an `els-cli.config` file:

```bash
[profiles.default]
  maxAPITries = 2
  [profiles.default.AccessKey]
    email = "clara@example.com"
    id = "MYACCESSKEYID"
    secretAccessKey = "MYSECRET"


[profiles.bob]
  maxAPITries = 3
  [profiles.bob.AccessKey]
    email = "suni@example.com"
    id = "ANOTHERACCESSKEYID"
    secretAccessKey = "ANOTHERSECRET"
    expiryDate = "2017-02-01T12:00:00Z"
```

## Prerequisites

### Create an Access Key
To create an Access Key for the first time, do the following:

    els-cli users EMAILADDRESS accessKey create [NUMDAYS]

e.g.

    els-cli users user@example.com accessKeys create

will create an Access Key which expires after a default period of 30 days, and

    els-cli users user@example.com accessKeys create 10

will create an Access Key which expires in 10 days.

### Install the Access Key in a config Profile

When you create an access key, you can make the els-cli use it by including it
in a profile saved to its config file, which is located at `~/.els/els-cli.toml`.

The data shown after successfully creating an Access Key can be written to the
config file, and defines the default profile - i.e. the profile that will be
used by els-cli if you don't specify a profile with --profile.

## Examples

### Create a new Fuel Charging Ruleset (Vendor role-holders only)

A Ruleset defines the rules which define how much Fuel a Feature consumes. There
is only ever one live Ruleset, but you can upload another and make it live
later. It is not possible to change a ruleset which is currently live. Instead,
upload a new ruleset and make it live.

To create a new ruleset, do:

    els-cli vendors VENDORID rulesets RULESETID create RULESETFILE

or

    echo RULESETFILE | els-cli vendors VENDORID rulesets RULESETID create

Where RULESETFILE is a file containg valid JSON defining the rules.

E.g. if the following ruleset is stored in file **ruleset_2016-02.json**

```json
{
  "rulesetDoc": {
    "rules": [
      {
        "id": "rule1",
        "title": "Laser Attack Pro Base Charge",
        "evaluator": {
            "operator": "A",
            "conditions": [
              {
                 "source"    : "feature",
                 "attribute" : "id",
                 "test"      : "is",
                 "value"     : "Laser Attack Pro"
              }
            ]
        },
        "actions": [
          {
             "target": "featureRate",
             "operation": "set",
             "currency": "GBP",
             "unit": "hour",
             "value": 0.01
          },
          {
             "target": "featureRate",
             "operation": "set",
             "currency": "EUR",
             "unit": "hour",
             "value": 0.011
          }
        ]
      }
    ]
  }
}

```
then you can create this ruleset by doing:

    els-cli vendors sharkSoft rulesets 2016-02 create ruleset_2016-02.json

or

    echo ruleset_2016-02.json | els-cli vendors sharkSoft rulesets 2016-02 create


**Note that a ruleset will not be used until you activate it** - See below...

### Activate an uploaded ruleset

This will cause the ruleset to be live, and fuel consumption prices will be
calculated with this ruleset immediately.

### Put a vendor (Elastic Licensing role-holders only)

To create or update a vendor, either prepare a file containing the JSON defining
the record, or

`els-cli vendor` *vendorID* `put` *jsonFile*

or via a pipe...

`cat` *jsonFile* `| els-cli vendor` *vendorID* `put`

# Making a Release of els-cli

(Site maintainers only)

We use [git flow](https://danielkummer.github.io/git-flow-cheatsheet/).

When preparing a new release, do the following:

1. Update the version in els-cli.go
2. Update the releases.md with details of the changes
3. Run `build.sh <version>`
4. Upload the artifacts from `_releases/<version>` to the
[github els-cli releases page](https://github.com/elasticlic/els-cli/releases).

