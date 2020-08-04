grad
===

Designed for using a go proxy+hostapd as an 802.1x RADIUS server for network authentication (or how to live without freeradius)

_grad is a fork and replacement for the concepts built from radiucal_

# purpose

This is a go proxy+hostapd setup that provides a very simple configuration to manage 802.1x authentication and management on a LAN.

Expectations:
* Running as a host/server on a variety of distriutions
* hostapd can do a lot with EAP and RADIUS as a service, this should serve as an exploration of these features
* Fully replace freeradius/radiucal+hostapd for 802.1x/AAA/etc.

## AAA

* Authentication (Your driver's license proves that you're the person you say you are)
* Authorization (Your driver's license lets you drive your car, motorcycle, or CDL)
* Accounting (A log shows that you've driven on these roads at a certain date)

## Goals

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
* Easy to deploy
* Utilize gitlab to perform user checking

## Proxy

Proxy that receives UDP packets and routes them along (namely to hostapd/another radius server)

the proxy:

* provides an approach to handle preauth, auth, postauth, and accounting actions
* can support user+mac filtering, packet logging, and access logging
* shares secrets with the backend hostapd server
* utilizes gitlab as a user management configuration system

# setup

## services

```
systemctl enable --now grad.service
```

## debugging

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
