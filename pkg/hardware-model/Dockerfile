# Copyright (c) 2018 Zededa, Inc.
# SPDX-License-Identifier: Apache-2.0

FROM lfedge/eve-alpine:0f2e0da38e30753c68410727a6cc269e57ff74f2 AS build
ENV BUILD_PKGS="git gcc linux-headers libc-dev make linux-pam-dev m4 findutils go util-linux-dev bash vim"
RUN eve-alpine-deploy.sh

# Install Go
ENV GOLANG_VERSION 1.21.5
ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
ADD https://dl.google.com/go/go${GOLANG_VERSION}.linux-amd64.tar.gz /tmp/
RUN tar -C /usr/local -xzf /tmp/go${GOLANG_VERSION}.linux-amd64.tar.gz

# Build hardware-model service
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o hardware-model ./cmd/hardware-model

# Final stage
FROM scratch
COPY --from=build /src/hardware-model /hardware-model
ENTRYPOINT ["/hardware-model"]