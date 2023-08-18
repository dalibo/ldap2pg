#!/bin/bash
# dev wrapper for conftest
exec go run ./cmd/ldap2pg "$@"
