FROM fedora:rawhide

RUN dnf update -y && dnf install -y golang go-bindata wget make fedora-packager

ARG RADIUCAL_VERSION
ENV RADIUCAL=radiucal-${RADIUCAL_VERSION}
ENV VERSION=${RADIUCAL_VERSION}

RUN wget https://cgit.voidedtech.com/radiucal/snapshot/${RADIUCAL}.tar.gz
RUN tar xf ${RADIUCAL}.tar.gz
RUN mv ${RADIUCAL} build/

RUN rpmdev-setuptree
RUN rmdir ~/rpmbuild/BUILD/
COPY *.spec ~/rpmbuild/SPECS/
RUN mv build/ ~/rpmbuild/BUILD

WORKDIR ~/rpmbuild/SPECS/

RUN rpmbuild -bb authem-utils.spec
RUN cp ~/rpmbuild/RPMS/x86_64/*.rpm /rpm/
