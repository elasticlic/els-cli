# els-cli
A commandline tool for interacting with the Elastic Licensing API.

## Terminology

**ELS** - *Elastic Licensing Service* - a cloud-based service which implements
the Elastic API - an HTTP REST Service.

**User Account** - Users are identified by *email address*, authenticated by
password. Most API calls must be made by a user, identified not by email address
and password but by *Access Key*.

**Elastic App** - An App which includes the **Elastic Client Library**.

**Access Key** - A key associated with a user, consisting of a public string
called the *AccessKeyID*, and a secret string called the *SecretAccessKey*.
Access Keys enable the user to keep their password secret. A user can change
their password without affecting any of their Access Keys, and can give another
person an Access Key without revealing their password. Access Keys are used to
**ELS-sign** an API call. A special API call lets a user create a new Access Key.
Access Keys can optionally expire, or be explicitly
deleted at any time. When an Elastic App is used, it can be invoked with
command-line arguments passing the Access Key to avoid the need to sign in on
startup.

**ELS-sign** - means to calculate and add a signature to an HTTP request so that
the ELS can be sure who made the API call.

**els-cli.toml** - a config file used by els-cli to provide certain values
and defaults. The config contains one or more *Profiles*. and should be placed
at the following location:

    ~/.els/els-cli.toml

The contents are [TOML](https://github.com/toml-lang/toml) - Tom's Obvious
Minimal Language

**Profile** - defines the Access Key to use to ELS-sign API calls, as well as
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

## Vendor Examples

### Create a new Fuel Charging Ruleset

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


**Note that a ruleset will not be used until you activate it.** - See below...

### Activate an uploaded ruleset

This will cause the ruleset to be live, and fuel consumption prices will be
calculated with this ruleset immediately.

### Put a vendor

To create or update a vendor, either prepare a file containing the JSON defining
the record, or

`els-cli vendor` *vendorID* `put` *jsonFile*

or via a pipe...

`cat` *jsonFile* `| els-cli vendor` *vendorID* `put`

