/*
els-cli is a commandline tool for issuing API calls to the Elastic Licensing
API (ELS API).

Use it to automate tasks with scripts, or to make changes which are not yet
implemented in the Elastic Licensing Dashboards. (For example, vendors can use
this to upload new Fuel Price Rulesets until the Ruleset editor is implemented
in the Vendor Dashboards).

Profiles

A Profile is a named collection of settings. Profiles are defined in the
els-cli config file - located at ~/.els/els-cli.config.

You can specify a profile with the option --profile or -p.

E.g.

  els-cli --profile="MyProfile" <rest of command>

If no profile is given, the "default" profile is expected.

Command Families

Some commands are broken down into families of subcommands, e.g. when they
relate to a specific entity.

E.g. all vendor manipulation commands require the specification of the vendorId
as follows:

  els-cli vendor VENDORID <rest of command>

To find out all the families of commands, use:

  els-cli -h

Content

When putting data to the ELS, you can either give the name of a file containing
the contents to put, or pipe the data into the command. E.g.:


  els-cli vendor VENDORID put <filename>

  or

  <filename> | els-cli vendor VENDORID put


The descriptions below will assume specifying the contents with a filename
argument.

Vendor Commands

els-cli vendor VENDORID get <filename>

Creates or updates the vendor, using the JSON contents of the given file, or
the data piped to the command

*/
package main
