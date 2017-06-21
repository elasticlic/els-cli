# Releases

## 0.0.3

* Updated to use sirupsen/logrus 1.0.0

----
## 0.0.2

New features:

### users `<email address>` accessKeys `<action>`

Where `<action>` is one of:
- **create** - Create a new API Access Key
- **delete** - Deled an API Access Key

### Vendors `<vendorID>` `<action>`

Where `<action>` is one of:
- **get** - (Elastic Licensing Employees only) Get details about a vendor
- **put** - (Elastic Licensing Employees only) - Update or Create a vendor

### Vendors `<vendorID>` rulesets `<action>`

Where `<action>` is one of:
- **put** - put a pricing ruleset
- **get** - get a pricing ruleset
- **activate** - activate a ruleset (i.e. so it is the ruleset used to generate pricing for Fuel)