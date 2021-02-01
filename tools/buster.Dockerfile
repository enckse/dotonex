FROM docker.io/debian:stable

RUN echo 'deb http://deb.debian.org/debian stable-backports main' > /etc/apt/sources.list.d/backports.list
RUN apt-get update && apt-get upgrade -y
RUN apt-get install -y build-essential golang-1.13-go git libnl-3-dev libnl-genl-3-dev libnl-route-3-dev libssl-dev
RUN update-alternatives --install /usr/bin/go go /usr/lib/go-1.13/bin/go 100
RUN mkdir /workdir
RUN mkdir /workdir/exported
WORKDIR /workdir
RUN git clone git://cgit.voidedtech.com/dotonex

ARG COMMIT
ARG RADIUSKEY
ARG SHAREDKEY
ARG GITLABFQDN
ARG SERVERREPO
ARG CERTKEY

RUN git -C dotonex checkout ${COMMIT}
WORKDIR /workdir/dotonex
RUN ./configure --go-flags '-buildmode=pie' --enable-gitlab --hostapd-certkey=${CERTKEY}  --radius-key=${RADIUSKEY} --shared-key=${SHAREDKEY} --gitlab-fqdn ${GITLABFQDN} --server-repository=${SERVERREPO}
RUN make
