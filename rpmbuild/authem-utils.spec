Name: authem-utils
Version: 2.2.1
Release: 1%{?dist}
Summary: radiucal admin tools

License: GPL-3
URL: https://cgit.voidedtech.com/radiucal

BuildRequires: git
BuildRequires: golang
BuildRequires: go-bindata

%description
Tools for administrating radiucal configurations for users

%build
echo $PWD
make authem-configurator authem-passwd

%files
/usr/bin/authem-configurator
/usr/bin/authem-passwd

%install
install -d $RPM_BUILD_ROOT/usr/bin/
install -Dm755 authem-configurator $RPM_BUILD_ROOT/usr/bin/authem-configurator
install -Dm755 authem-passwd $RPM_BUILD_ROOT/usr/bin/authem-passwd
