# Copyright 2022 Authors of spidernet-io
# SPDX-License-Identifier: Apache-2.0


ARG BASE_IMAGE=ghcr.io/spidernet-io/netknife/netknife-base:52b5798da79b82948067b911f341853d1a04d68f@sha256:c939188a6c5c8df6415b8fda55be902dad877e4b1d75076be6cbf807a65a345c
ARG GOLANG_IMAGE=docker.io/library/golang:1.17.8@sha256:f675106e44f205a7284e15cd75c41b241329f3c03ac30b0ba07b14a6ea7c99d9

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
        build-bin

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
COPY --from=builder /src/scripts   /scripts

CMD ["/scripts/http/start"]
