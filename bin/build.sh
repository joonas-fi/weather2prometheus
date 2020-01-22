#!/bin/bash -eu

source /build-common.sh

BINARY_NAME="weather"
COMPILE_IN_DIRECTORY="cmd/weather"

standardBuildProcess

buildstep packageLambdaFunction
