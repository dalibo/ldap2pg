#!/bin/bash

set -eu

adduser() {
	samba-tool user add --random-password "$@"
}

# a* for alter
adduser alain
# UTF-8 case
adduser alizée
# SQL keyword
adduser alter

# c* for creation
adduser corinne
adduser charles
# Clothile has a capital letter. This is a test for case insensitivity
adduser Clothilde

# Blacklisted
adduser postgres

samba-tool group add readers
samba-tool group addmembers readers alain,corinne,postgres

samba-tool group add writers
samba-tool group addmembers writers alizée,charles

samba-tool group add owners
samba-tool group addmembers owners alter,Clothilde
