FROM golang:1.16-buster

RUN apt-get update && apt-get upgrade -y
RUN apt-get install -y build-essential git libnl-3-dev libnl-genl-3-dev libnl-route-3-dev libssl-dev
RUN mkdir /workdir
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
RUN ./configure --build-mode gitlab --hostapd-certkey=${CERTKEY} --radius-key=${RADIUSKEY} --shared-key=${SHAREDKEY} --gitlab-fqdn ${GITLABFQDN} --server-repository=${SERVERREPO}
RUN make
RUN make check
