#!/bin/sh

exec tail -f /var/log/messages > /dev/tty1
