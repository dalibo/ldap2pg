#!/bin/bash -eux

apt-get update -y
apt-get install -y libsasl2-modules
exec /container/tool/run --copy-service $@
