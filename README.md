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

    els-cli user EMAILADDRESS accessKey create

### Install the Access Key in a config Profile

ABHERE

## Vendor Examples


### Put a vendor

To create or update a vendor, either prepare a file containing the JSON defining
the record, or

`els-cli vendor` *vendorID* `put` *jsonFile*

or via a pipe...

`cat` *jsonFile* `| els-cli vendor` *vendorID* `put`

