#!/bin/sh

msg() {
    # bold
    printf "\033[1m=> $@\033[m\n"
}

msg "Creating Mountpoint"
mkdir -p /mnt/system

msg "Mounting Gizmo"
mount -o sync /dev/sda1 /mnt/system

msg "Copying Firmware"
cp /usr/share/gizmo/gss/firmware.uf2 /mnt/system/

msg "Releasing Gizmo"
umount /mnt/system
