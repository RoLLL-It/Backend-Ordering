#!/bin/bash
set -e
export GOFLAGS=-mod=vendor
go build -o exec .
