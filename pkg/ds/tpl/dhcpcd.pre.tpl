#!/bin/sh

ip link add name br0 type bridge
ip link set dev br0 up

ip link set dev wlan0 master br0
ip link set dev eth0 master br0
ip link set dev eth0 up
