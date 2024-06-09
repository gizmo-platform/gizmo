#!/bin/sh

# Get a fresh void image
wget -qO void.img.xz http://repo-fastly.voidlinux.org/live/current/void-rpi-aarch64-musl-20240314.img.xz
xz -d void.img.xz

# Setup and mount
LDEV=$(losetup --show -f -P void.img)
mkdir -p /mnt/target
mount ${LDEV}p2 /mnt/target
for _fs in dev proc sys; do
	mount --rbind "/$_fs" "/mnt/target/$_fs"
	mount --make-rslave "/mnt/target/$_fs"
done
touch /mnt/target/etc/resolv.conf
mount -o bind /etc/resolv.conf /mnt/target/etc/resolv.conf

# Install the Gizmo tools
cp dist/gizmo_linux_arm64/gizmo /mnt/target/usr/local/bin/gizmo
cd /mnt/target || exit 1
chroot /mnt/target /usr/local/bin/gizmo ds install
echo root:gizmo | chpasswd -c SHA512 -R /mnt/target

# Backout, unmount, compress
cd - || exit 1
umount -R /mnt/target
losetup -d $LDEV
mv void.img driver-station.img
sync
zip driver-station.zip driver-station.img
