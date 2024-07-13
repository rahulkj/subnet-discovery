#!/bin/bash

if [[ "${1}" == "-r" ]]; then
    echo "Generating release builds"
    goreleaser release --skip=announce,publish,validate --clean
fi

if [[ "${1}" == "-l" ]]; then
    OS=$(uname)
    echo "Generating local build for your $OS"
    go build
fi