#!/bin/sh

# Ensure that $PWD is the root of this repository
SCRIPT_PATH=`dirname $0`
cd "${SCRIPT_PATH}/.."

if gofmt -l . 2>&1 | read line; then
  # There were errors.  Re-run gofmt to print them out into the test log.
  echo "The following files are not formatted correctly according to gofmt:"
  gofmt -l .
  exit 1
fi
