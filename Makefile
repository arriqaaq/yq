.PHONY : build run fresh test clean pack-releases

BIN := yq

HASH := $(shell git rev-parse --short HEAD)
COMMIT_DATE := $(shell git show -s --format=%ci ${HASH})
BUILD_DATE := $(shell date '+%Y-%m-%d %H:%M:%S')
VERSION := ${HASH} (${COMMIT_DATE})
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))


build:
	go build -o ${BIN} -ldflags="-X 'main.buildVersion=${VERSION}' -X 'main.buildDate=${BUILD_DATE}'" $(PROJECT_DIR)/main.go

fresh: clean build

test:
	go test

clean:
	go clean
	- rm -f ${BIN}