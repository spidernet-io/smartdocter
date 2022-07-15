# Copyright 2022 Authors of spidernet-io
# SPDX-License-Identifier: Apache-2.0

ARG BASE_IMAGE=ghcr.io/spidernet-io/smartdocter/smartdocter-agent-base:14bc0e1babc419f3f233577c296f3a23af855529@sha256:ba1dd78c1ed93967c48ea56187ecf64529f6d5d63308cb6e1389e710f8490445
ARG GOLANG_IMAGE=docker.io/library/golang:1.18.2@sha256:04fab5aaf4fc18c40379924674491d988af3d9e97487472e674d0b5fd837dfac

# TARGETARCH is an automatic platform ARG enabled by Docker BuildKit.
# like amd64 arm64
ARG TARGETARCH

#======= build bin ==========
FROM --platform=${BUILDPLATFORM} ${GOLANG_IMAGE} as builder

ARG TARGETOS
ARG TARGETARCH
ARG RACE
ARG NOSTRIP
ARG NOOPT
ARG QUIET_MAKE

COPY . /src
WORKDIR /src
RUN  make GOARCH=${TARGETARCH}   \
        RACE=${RACE} NOSTRIP=${NOSTRIP} NOOPT=${NOOPT} QUIET_MAKE=${QUIET_MAKE} \
        DESTDIR_BIN=/tmp/install/${TARGETOS}/${TARGETARCH}/bin \
        build_smartdocter_agent_bin

RUN chmod -R +x /src/scripts/*


#====== release image =======

FROM ${BASE_IMAGE}

LABEL maintainer="maintainer@spidernet-io"

# TARGETOS is an automatic platform ARG enabled by Docker BuildKit.
ARG TARGETOS
# TARGETARCH is an automatic platform ARG enabled by Docker BuildKit.
ARG TARGETARCH

ARG GIT_COMMIT_VERSION
ENV GIT_COMMIT_VERSION=${GIT_COMMIT_VERSION}
ARG GIT_COMMIT_TIME
ENV GIT_COMMIT_TIME=${GIT_COMMIT_TIME}
ARG VERSION
ENV VERSION=${VERSION}

COPY --from=builder /tmp/install/${TARGETOS}/${TARGETARCH}/bin/*   /usr/bin/
COPY --from=builder /src/scripts/smartdocter-agent   /scripts

CMD ["/usr/bin/serve-hostname"]
