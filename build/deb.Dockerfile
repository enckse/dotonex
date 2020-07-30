FROM debian:sid

RUN apt-get update && apt-get -y upgrade
RUN apt-get install -y wget golang debhelper git go-bindata build-essential make

ARG RADIUCAL_VERSION
ENV RADIUCAL=radiucal-${RADIUCAL_VERSION}
ENV VERSION=${RADIUCAL_VERSION}

RUN wget https://cgit.voidedtech.com/radiucal/snapshot/${RADIUCAL}.tar.gz
RUN tar xf ${RADIUCAL}.tar.gz
RUN mv ${RADIUCAL} build/
COPY debian build/debian
WORKDIR build
RUN dpkg-buildpackage -us -uc --build=binary
RUN cp ../*.deb /deb/
