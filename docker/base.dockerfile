################################################################################
FROM golang:1.15 as build

ENV GOPROXY=https://proxy.golang.org
ARG PROTOC_VERSION=3.13.0
RUN apt-get -y -q -o=Dpkg::Use-Pty=0 update && \
    apt-get -y -q -o=Dpkg::Use-Pty=0 install --no-install-recommends unzip && \
    apt-get -y -q -o=Dpkg::Use-Pty=0 autoremove && \
    rm -rf /var/lib/apt/lists/* && \
    curl -fsSL -o $GOPATH/bin/protoc.zip https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip &&\
    unzip -q -o -j $GOPATH/bin/protoc.zip -d $GOPATH/bin bin/protoc && \
    unzip -q -o $GOPATH/bin/protoc.zip -d $GOPATH 'include/*' && \
    rm $GOPATH/bin/protoc.zip

################################################################################
FROM ubuntu:20.04 as runtime

RUN apt-get -y -q -o=Dpkg::Use-Pty=0 update && \
    apt-get -y -q -o=Dpkg::Use-Pty=0 install --no-install-recommends \
        locales \
        ca-certificates \
        gettext-base \
        wget && \
    locale-gen en_US.UTF-8 && \
    update-locale LC_ALL=en_US.UTF-8 LANG=en_US.UTF-8 && \
    apt-get -y -q -o=Dpkg::Use-Pty=0 remove unattended-upgrades ubuntu-release-upgrader-core update-notifier && \
    (apt-get -y -q -o=Dpkg::Use-Pty=0 remove popularity-contest || true) && \
    (apt-get -y -q -o=Dpkg::Use-Pty=0 purge apport || true) && \
    (apt-get -y -q -o=Dpkg::Use-Pty=0 purge snapd || true) && \
    apt-get -y -q -o=Dpkg::Use-Pty=0 autoremove && \
    groupadd -g 1001 -r discordbot && \
    useradd -u 1001 -g 1001 -d /home/discordbot -m -r -s /bin/bash discordbot && \
    wget -q -O honeytail https://honeycomb.io/download/honeytail/v1.2.0/honeytail-linux-amd64 && \
    echo 'd830774a620f6ecc4b39898bd5349f75fc86f4852314f42e0ac80c0e8b735677  honeytail' | sha256sum -c && \
    chmod 755 ./honeytail && \
    mv ./honeytail /usr/local/bin/


ENV LANG='en_US.UTF-8' LANGUAGE='en_US:en' LC_ALL='en_US.UTF-8' PATH=/usr/local/bin:$PATH
VOLUME /tmp