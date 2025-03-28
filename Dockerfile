# syntax=docker/dockerfile:1

# ======== Build stage ========
FROM golang:1.24-alpine AS build
RUN apk add make

ARG VERSION
ARG BUILD_TIME
ARG GO_BUILD_TAGS=

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Set build environment variables
ENV BIN_OUT_DIR="/bin" \
	GOOS=linux \
	GO_BUILD_TAGS="${GO_BUILD_TAGS} nogpu"

# Build
RUN make build

# ======== Final stage ========
FROM scratch

ARG VERSION
ARG BUILD_TIME

LABEL org.opencontainers.image.title="mqttop" \
	org.opencontainers.image.vendor="lone-faerie" \
	org.opencontainers.image.license="AGPL-3.0" \
	org.opencontainers.image.version="${VERSION}" \
	org.opencontainers.image.created="${BUILD_TIME}" \
	org.opencontainers.image.source="https://github.com/lone-faerie/mqttop"

WORKDIR /app

COPY --link --from=build /bin/mqttop /app/mqttop

ENV MQTTOP_CONFIG_PATH=/config/config.yml

ENTRYPOINT ["/app/mqttop", "run"]
