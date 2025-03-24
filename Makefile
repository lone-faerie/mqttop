.PHONY: all clean build run docker docker-build docker-build-gpu

BIN_OUT_DIR?=bin
BIN_PATH=${BIN_OUT_DIR}/mqttop

PACKAGE=github.com/lone-faerie/mqttop
VERSION?=$(shell git describe --always --tags)

GO_BUILD_TAGS?=
comma:=,
empty:=
space:=$(empty) $(empty)
GO_BUILD_TAGS:=$(subst $(space),$(comma),$(strip $(GO_BUILD_TAGS)))

LDFLAGS:=-X '${PACKAGE}/internal/build.version=${VERSION}' \
	 -X '${PACKAGE}/internal/build.pkg=${PACKAGE}'

all: clean build ## Build binary

clean: ## Clean output directory
	go clean
	rm ${BIN_OUT_DIR}/*

build: ## Build binary
	go build -tags ${GO_BUILD_TAGS} -ldflags="${LDFLAGS}" -o ${BIN_PATH} ./cmd

debug: ## Build binary with 'debug' tag
	echo $(subst $(space),$(comma),$(strip $(GO_BUILD_TAGS) debug))
	go build -tags $(subst $(space),$(comma),$(strip $(GO_BUILD_TAGS) debug)) -ldflags="${LDFLAGS}" -o ${BIN_PATH} ./cmd

run: ## Build and run binary
	go run -tags ${GO_BUILD_TAGS} -ldflags="${LDFLAGS}" ./cmd

docker: docker-build docker-build-gpu ## Build both docker images

docker-build: ## Build docker image without GPU support
	docker buildx build \
		--build-arg VERSION=${VERSION} \
		--tag mqttop \
		-f Dockerfile \
		.

docker-build-gpu: ## Build docker image with GPU support
	docker buildx build \
		--build-arg VERSION=${VERSION} \
		--tag mqttop:gpu \
		-f Dockerfile.gpu \
		.
