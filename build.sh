#!/bin/bash


VERSION="1.0"
GITHASH=$(git rev-parse HEAD | sed 's/\(^........\).*/\1/' | tr '[a-z]' '[A-Z]')
BUILDSTAMP=$(date -u '+%Y%m%d%H%M%S')

LDFLAGS="-X check_gobw/config.GITHASH=${GITHASH} -X check_gobw/config.BUILDSTAMP=${BUILDSTAMP} -X check_gobw/config.VERSION=${VERSION}"

go build -ldflags "${LDFLAGS}" -o check_gobw
GOARCH=386 go build -ldflags "${LDFLAGS}" -o check_gobw32
