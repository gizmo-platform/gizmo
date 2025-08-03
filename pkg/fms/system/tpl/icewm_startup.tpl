#!/bin/sh

pipewire &
volctl &
xterm -fs 22 -e gizmo-sysconf &
