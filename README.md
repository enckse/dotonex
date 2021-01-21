dotonex
===

Designed for using a go proxy+hostapd as an 802.1x RADIUS server for network authentication (or how to live without freeradius).
This is the 4th generation implementation and deployment of custom configurations in this area

# purpose

This is a go proxy+hostapd setup that provides a very simple configuration to manage 802.1x authentication and management on a LAN.

Expectations:
* Running on Linux as a host/server
* hostapd can do a lot with EAP and RADIUS as a service, this should serve as an exploration of these features
* Fully replace freeradius for 802.1x/AAA/etc.

## AAA

* Authentication (Your driver's license proves that you're the person you say you are)
* Authorization (Your driver's license lets you drive your car, motorcycle, or CDL)
* Accounting (A log shows that you've driven on these roads at a certain date)

## Goals

* Utilize a backend service to authorize users
* Support a port-restricted LAN (+wifi) in a controlled, physical area
* Provide a singular authentication strategy for supported clients using peap+mschapv2 (no CA validation).
* Windows 10
* Linux (any supported modern versions of NetworkManager or direct `wpa_supplicant` usage)
* Android 7+
* Map authenticated user+MAC combinations to specific VLANs
* Support MAC-based authentication (bypass) for systems that can not authenticate themselves
* Integrate with a variety of network equipment
* Avoid client-issued certificates (and management)
* Centralized configuration file
* As few open endpoints as possible on the radius server (only open ports 1812 and 1813 for radius)

**These goals began with the usage of freeradius and continue to be vital to our operation**

## Proxy

dotonex is a go proxy that receives UDP packets and routes them along (namely to hostapd/another radius server)

the proxy:

* provides a modularized/plugin approach to handle preauth, auth, postauth, and accounting actions
* can support user+mac filtering, logging, debug output, and simple stat output via plugins
* provides a cut-in for more plugins
* overrides the concept of "radius_clients" as all will have to have a single shared secret

# setup

## build

Requires base development tools and go to build, it will build the whole stack and a local instance
of hostapd as certain flags are not always set for each distribution.

to deploy and utilize the utilize default backend:
- gitlab auth for user names (mapping to gitlab user token's)
- repository containing simplistic user+MAC combinations and VLAN definitions/configurations

```
./configure \
    -radius-key networkkey \
    -shared-key clientkey \
    -enable-gitlab \
    -gitlab-fqdn gitlab.example.com \
    -server-repository /path/to/repo
```

to build the solution

```
make
```

and finally to install it
```
sudo make install
```

## configure

It is **suggested** to initially run the daemon by-hand to verify setup (not required)
```
sudo dotonex-daemon
```

Also make sure to enable the necessary services
```
sudo systemctl enable dotonex.service
```

## debugging

### builds

Builds can use the `-development` boolean to disable install setups/requirements for developmental efforts

```
./configure -development
```

and then to build and run tests

```
make
```

### remotely

this requires that:
* the radius server is configured to listen/accept on the given ip below
* MAC is formatted as 00:11:22:aa:bb:cc
* `eapol_test` is installed

setup a test config
```
vim test.conf
---
network={
        key_mgmt=WPA-EAP
        eap=PEAP
        identity="<vlan.user>"
        password="<password>"
        phase2="autheap=MSCHAPV2"
}
```

to run
```
eapol_test -a <radius_server_ip> -c test.conf -s <secret_key> -M <mac>
```
