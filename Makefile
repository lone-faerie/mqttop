.PHONY: all build

BIN_OUT_DIR?=bin
BIN_PATH=${BIN_OUT_DIR}/mqttop

PACKAGE=github.com/lone-faerie/mqttop
VERSION?=0.0.2

GO_BUILD_TAGS?=
comma:=,
empty:=
space:=$(empty) $(empty)
GO_BUILD_TAGS:= $(subst $(space),$(comma),$(strip $(GO_BUILD_TAGS)))

LDFLAGS:=-X '${PACKAGE}/internal/build.version=${VERSION}' \
	 -X '${PACKAGE}/internal/build.pkg=${PACKAGE}'

all: build

build:
	go build -tags ${GO_BUILD_TAGS} -ldflags="${LDFLAGS}" -o ${BIN_PATH} ./cmd

clean:
	go clean
	rm ${BIN_PATH}

run:
	go run -ldflags="${LDFLAGS}" ./cmd

docker-build:
	docker buildx build --tag mqttop .
