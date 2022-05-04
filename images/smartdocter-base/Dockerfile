# docker buildx build -t testbase:latest --platform=linux/arm64,linux/amd64 --output type=docker .

# Copyright 2022 Authors of spidernet-io
# SPDX-License-Identifier: Apache-2.0

ARG GOLANG_IMAGE=docker.io/library/golang:1.17.8@sha256:f675106e44f205a7284e15cd75c41b241329f3c03ac30b0ba07b14a6ea7c99d9
ARG UBUNTU_IMAGE=docker.io/library/ubuntu:21.10@sha256:14d8cab8d7bd72a01e225cd85bf37bbf08bf88115769316fa86c776df4ea5fed
ARG CADDY_IMAGE=caddy:2.5.0@sha256:e42e57a219aa11138f3f147fb0c80b364c57fb063f85568664136ca9f7dd0048

#========= build gops ===============
FROM --platform=${BUILDPLATFORM} ${GOLANG_IMAGE} as gops-cni-builder
COPY /build-gops.sh /build-gops.sh

ARG USE_PROXY_SOURCE
#use alibaba debian source
COPY /sources.list /sources.list
RUN [ "$USE_PROXY_SOURCE" != "true" ] && exit 0 ; \
      rm -f /etc/apt/sources.list ; \
      cp /sources.list /etc/apt/sources.list ;  \
      apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 648ACFD622F3D138

#with debian 11 os
RUN apt-get update && apt-get install -y binutils-aarch64-linux-gnu binutils-x86-64-linux-gnu

# build-gops.sh will build both archs at the same time
WORKDIR /
RUN chmod +x /build-gops.sh && /build-gops.sh


#============
FROM ${CADDY_IMAGE} as caddy

#========== root image ==============
FROM ${UBUNTU_IMAGE} as rootfs
COPY /install-others.sh /install-others.sh


#fix warning: debconf: delaying package configuration, since apt-utils is not installed
ARG DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get upgrade -y

WORKDIR /

ARG TARGETPLATFORM
COPY --from=gops-cni-builder /out/${TARGETPLATFORM}/bin/gops /bin/gops

COPY --from=caddy /usr/bin/caddy /usr/bin/caddy

RUN chmod +x /install-others.sh && /install-others.sh && rm -f /install-others.sh


#============
FROM scratch
LABEL maintainer="maintainer@spidernet-io"
COPY --from=rootfs / /
