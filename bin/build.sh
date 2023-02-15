#!/bin/bash -eu


build-go-project.sh --directory=cmd/weather/ --binary-basename=weather

# using this rest for packageLambdaFunction

source /build-common.sh

BINARY_NAME="weather"
COMPILE_IN_DIRECTORY="cmd/weather"

buildstep packageLambdaFunction
