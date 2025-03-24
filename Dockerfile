# syntax=docker/dockerfile:1

# ======== Build stage ========
FROM golang:1.24 AS build

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Set build environment variables
ENV BIN_OUT_DIR="/bin" \
	GOOS=linux
	GO_TAGS=debug

# Build
RUN make build

# ======== Final stage ========
FROM debian

WORKDIR /app

COPY --link --from=build /bin/mqttop /app/mqttop

ENV MQTTOP_CONFIG_PATH=/config/config.yml

ENTRYPOINT ["/app/mqttop", "run"]
