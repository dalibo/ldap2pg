#!/bin/bash

set -eu

adduser() {
	samba-tool user add --random-password --mail-address="$1@bridoulou.fr" "$1"
}

# s* fro superusers
adduser solene
adduser samuel

samba-tool group add dba
samba-tool group addmembers dba solene,samuel
