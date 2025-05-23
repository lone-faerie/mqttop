.PHONY: all clean build run docker docker-build docker-build-gpu

BIN_OUT_DIR?=bin
BIN_PATH=${BIN_OUT_DIR}/mqttop

PACKAGE=github.com/lone-faerie/mqttop
VERSION?=$(shell git describe --always --tags)
BUILD_TIME?=$(subst $(space),T,$(shell date --rfc-3339=seconds))

GO_BUILD_TAGS?=
comma:=,
empty:=
space:=$(empty) $(empty)
GO_BUILD_TAGS:=$(subst $(space),$(comma),$(strip $(GO_BUILD_TAGS)))

LDFLAGS:=-X '${PACKAGE}/internal/build.pkg=${PACKAGE}' \
	 -X '${PACKAGE}/internal/build.version=${VERSION}' \
	 -X '${PACKAGE}/internal/build.buildTime=${BUILD_TIME}'

GO_BUILD_FLAGS=-ldflags="${LDFLAGS}"
ifneq ($(strip ${GO_BUILD_TAGS}), $(empty))
	GO_BUILD_FLAGS+=-tags ${GO_BUILD_TAGS}
endif

all: clean build ## Build binary

clean: ## Clean output directory
	go clean
	rm -f ${BIN_OUT_DIR}/*

build: ## Build binary
	go build ${GO_BUILD_FLAGS} -o ${BIN_PATH} .

install: clean build ## Build and install binary
	sudo cp ${BIN_PATH} /usr/local/bin/mqttop
	@if [ -x /bin/bash ]; then\
		sudo mqttop completion bash > /etc/bash_completion.d/mqttop;\
	fi
	@if [ -x /bin/zsh ]; then\
		sudo mqttop completion zsh > "${fpath[1]}/_mqttop";\
	fi
	@if [ -x /bin/fish ]; then\
		sudo mqttop completion fish > ~/.config/fish/completions/mqttop.fish;\
	fi

debug: ## Build binary with 'debug' tag
	go build -tags $(subst $(space),$(comma),$(strip $(GO_BUILD_TAGS) debug)) -ldflags="${LDFLAGS}" -o ${BIN_PATH} ./

cover: ## Build binary for coverage
	go build -cover ${GO_BUILD_FLAGS} -o ${BIN_PATH} .

run: ## Build and run binary
	go run ${GO_BUILD_FLAGS} .

docker: docker-build docker-build-gpu ## Build both docker images

docker-build: ## Build docker image without GPU support
	docker buildx build \
		--build-arg VERSION=${VERSION} \
		--build-arg BUILD_TIME=${BUILD_TIME} \
		--tag mqttop \
		-f Dockerfile \
		.

docker-build-gpu: ## Build docker image with GPU support
	docker buildx build \
		--build-arg VERSION=${VERSION} \
		--build-arg BUILD_TIME=${BUILD_TIME} \
		--tag mqttop:gpu \
		-f Dockerfile.gpu \
		.

docker-debug:
	docker buildx build \
		--build-arg VERSION=${VERSION} \
		--build-arg BUILD_TIME=${BUILD_TIME} \
		--build-arg GO_BUILD_TAGS=debug \
		--tag mqttop:development \
		-f Dockerfile \
		.

docker-debug-gpu:
	docker buildx build \
		--build-arg VERSION=${VERSION} \
		--build-arg BUILD_TIME=${BUILD_TIME} \
		--build-arg GO_BUILD_TAGS=debug \
		--tag mqttop:development-gpu \
		-f Dockerfile.gpu \
		.

# From https://github.com/prometheus/procfs
%/.unpacked: %.ttar
	@echo ">> extracting fixtures $*"
	./ttar -C $(dir $*) -x -f $*.ttar
	touch $@

# From https://github.com/prometheus/procfs
update_fixtures:
	rm -vf testdata/fixtures/.unpacked
	./ttar -c -f testdata/fixtures.ttar -C testdata/ fixtures/
