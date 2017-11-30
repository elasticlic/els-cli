#!/bin/bash
#
# Builds release versions of els-cli for upload to a github release. Builds are
# placed in directory _releases/<version>
#
# Usage: build.sh <version>
#
# e.g. build.sh 1.7.4

set -e

function extension {
    if [ "$GOOS" == "windows" ]; then
      echo ".exe"
    fi
}

function build {
    GOOS=$1
    GOARCH=$2

    echo Building "$GOOS" "$GOARCH"

    # Note: CGO_ENABLED=0 tells the go toolchain to try to avoid pulling in
    # dependencies which would require the executable to be dynamic (i.e.
    # relying on dynamic system libraries).
    #
    # ldflags '-s' omits the symbol table. (Stripping the executable is not
    # recommended).
    GOOS=$1 GOARCH=$2 CGO_ENABLED=0 go build -ldflags '-s'

    if [ $? -ne 0 ]; then
        echo "Build failed ${GOOS}, ${GOARCH}"
        exit -1
    fi

    OUTPUT=els-cli$(extension)
    DST_DIR="_releases/${VERSION}"
    DST="$DST_DIR"/els-cli-v"$VERSION"-"$GOOS"-"$GOARCH"$(extension)

    mkdir -p "$DST_DIR"
    mv -f "$OUTPUT" "$DST"
}


VERSION=$1
SCRIPT=$(basename "$0")

if [ "$VERSION" == "" ]; then
    echo Usage: "$SCRIPT" "<version>"
    exit -1
fi

if ! [[ $VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo Version must be of the format \'A.B.C\' where \'A\', \'B\' and \'C\' are all positive integers.
    exit -1
fi

build linux amd64
build windows amd64
build darwin amd64
