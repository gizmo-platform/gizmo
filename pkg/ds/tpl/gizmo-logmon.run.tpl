#!/bin/sh

exec tail -f /var/log/socklog/*/current > /dev/tty1
